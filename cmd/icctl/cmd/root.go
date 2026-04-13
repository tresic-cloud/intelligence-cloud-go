package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// environmentBaseURLs maps well-known environment names to their default API
// base URLs. Custom environments require an explicit --base-url.
var environmentBaseURLs = map[string]string{
	"production": "https://api.intelligence.cloud",
	"staging":    "https://api.staging.intelligence.cloud",
	"dev":        "https://api.dev.intelligence.cloud",
}

// ProfileLoader is the interface that root command uses to resolve profile
// configuration. W3-1's profile.Store will satisfy this interface at merge.
type ProfileLoader interface {
	// Load returns the non-secret profile fields for the given name.
	// Returns an error if the profile does not exist.
	Load(name string) (*ProfileConfig, error)

	// CurrentProfile returns the name of the current default profile
	// from config.yaml's current_profile key.
	CurrentProfile() (string, error)
}

// SecretLoader is the interface that root command uses to load tokens from
// the keychain or fallback credentials file. W3-1's profile.SecretStore
// will satisfy this interface at merge.
type SecretLoader interface {
	// LoadToken returns the bearer token for the given profile name.
	// Returns an empty string and nil error if no token is stored.
	LoadToken(name string) (string, error)
}

// ProfileConfig holds the non-secret fields of a profile, matching the
// config.yaml schema (cli-schema.md §Configuration files).
type ProfileConfig struct {
	Name          string
	Environment   string
	BaseURL       string
	DefaultOutput string
}

// NewRootCmd creates the root cobra command with all global flags registered
// and viper bindings configured. The profileLoader and secretLoader may be
// nil for testing; if nil, profile resolution is skipped.
func NewRootCmd(profileLoader ProfileLoader, secretLoader SecretLoader) *cobra.Command {
	v := viper.New()

	rootCmd := &cobra.Command{
		Use:   "icctl",
		Short: "Intelligence Cloud CLI",
		Long:  "icctl is the command-line interface for the Intelligence Cloud platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return resolveState(cmd, v, profileLoader, secretLoader)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Register global flags per cli-schema.md.
	pf := rootCmd.PersistentFlags()
	pf.String("profile", "default", "Profile name to use")
	pf.String("environment", "", "Environment (production/staging/dev/custom)")
	pf.String("base-url", "", "API base URL (overrides environment mapping)")
	pf.String("token", "", "Bearer token (one-off; does not update profile)")
	pf.StringP("output", "o", "table", "Output format (table|json)")
	pf.BoolP("yes", "y", false, "Bypass destructive-verb confirmation")
	pf.String("log-level", "warn", "Log level (debug|info|warn|error)")
	pf.String("request-id", "", "Force X-Request-Id header value")
	pf.String("timeout", "30s", "Per-call timeout")

	// Bind environment variables with ICCTL_ prefix.
	_ = v.BindPFlag("profile", pf.Lookup("profile"))
	_ = v.BindPFlag("environment", pf.Lookup("environment"))
	_ = v.BindPFlag("base_url", pf.Lookup("base-url"))
	_ = v.BindPFlag("token", pf.Lookup("token"))
	_ = v.BindPFlag("output", pf.Lookup("output"))
	_ = v.BindPFlag("assume_yes", pf.Lookup("yes"))
	_ = v.BindPFlag("log_level", pf.Lookup("log-level"))
	_ = v.BindPFlag("request_id", pf.Lookup("request-id"))
	_ = v.BindPFlag("timeout", pf.Lookup("timeout"))

	v.SetEnvPrefix("ICCTL")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Add subcommands.
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewProfileCmd(profileLoader, secretLoader))

	return rootCmd
}

// resolveState builds the CLI State from the 4-layer precedence and stores
// it in the command's context.
func resolveState(cmd *cobra.Command, v *viper.Viper, profileLoader ProfileLoader, secretLoader SecretLoader) error {
	// Layer 1: built-in defaults are already set via flag defaults.
	// Layer 2: profile values (loaded if profileLoader is available).
	var profileCfg *ProfileConfig

	profileName := resolveString(cmd, v, "profile", "ICCTL_PROFILE", "default")

	if profileLoader != nil {
		// If --profile was not explicitly set but there's a current_profile in config, use it.
		if !cmd.Flags().Changed("profile") && os.Getenv("ICCTL_PROFILE") == "" {
			if current, err := profileLoader.CurrentProfile(); err == nil && current != "" {
				profileName = current
			}
		}

		cfg, err := profileLoader.Load(profileName)
		if err != nil {
			// If profile not found, only error if it was explicitly requested.
			if cmd.Flags().Changed("profile") || os.Getenv("ICCTL_PROFILE") != "" {
				return fmt.Errorf("profile %q not found: %w", profileName, err)
			}
			// Otherwise fall through with no profile.
		} else {
			profileCfg = cfg
		}
	}

	// Resolve each setting with 4-layer precedence: flag > env > profile > default.
	environment := resolveWithProfile(cmd, v, "environment", "ICCTL_ENVIRONMENT", "", profileCfg, func(p *ProfileConfig) string { return p.Environment })
	baseURL := resolveWithProfile(cmd, v, "base-url", "ICCTL_BASE_URL", "", profileCfg, func(p *ProfileConfig) string { return p.BaseURL })
	output := resolveWithProfile(cmd, v, "output", "ICCTL_OUTPUT", "table", profileCfg, func(p *ProfileConfig) string { return p.DefaultOutput })
	token := resolveString(cmd, v, "token", "ICCTL_TOKEN", "")
	logLevel := resolveString(cmd, v, "log-level", "ICCTL_LOG_LEVEL", "warn")
	requestID := resolveString(cmd, v, "request-id", "ICCTL_REQUEST_ID", "")
	timeoutStr := resolveString(cmd, v, "timeout", "ICCTL_TIMEOUT", "30s")
	assumeYes := resolveYes(cmd, v)

	// If base-url is still empty, resolve from environment name.
	if baseURL == "" && environment != "" {
		if u, ok := environmentBaseURLs[environment]; ok {
			baseURL = u
		}
	}

	// If token is still empty and we have a secretLoader, try to load from profile.
	if token == "" && secretLoader != nil && profileName != "" {
		if t, err := secretLoader.LoadToken(profileName); err == nil {
			token = t
		}
	}

	// Parse timeout duration.
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid --timeout %q: %w", timeoutStr, err)
	}

	// Validate output format.
	switch output {
	case "table", "json":
		// valid
	default:
		return fmt.Errorf("invalid --output %q: must be table or json", output)
	}

	// Validate log level.
	switch logLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return fmt.Errorf("invalid --log-level %q: must be debug, info, warn, or error", logLevel)
	}

	state := &State{
		ProfileName:          profileName,
		EffectiveBaseURL:     baseURL,
		EffectiveToken:       token,
		EffectiveOutput:      output,
		EffectiveEnvironment: environment,
		AssumeYes:            assumeYes,
		LogLevel:             logLevel,
		RequestID:            requestID,
		Timeout:              timeout,
	}

	cmd.SetContext(WithState(cmd.Context(), state))
	return nil
}

// resolveString resolves a string value with precedence: flag > env > default.
func resolveString(cmd *cobra.Command, _ *viper.Viper, flagName, envVar, defaultVal string) string {
	// 1. Explicit flag.
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetString(flagName)
		return val
	}

	// 2. Environment variable.
	if envVal := os.Getenv(envVar); envVal != "" {
		return envVal
	}

	// 3. Default.
	return defaultVal
}

// resolveWithProfile resolves a value with 4-layer precedence:
// flag > env > profile > default.
func resolveWithProfile(cmd *cobra.Command, _ *viper.Viper, flagName, envVar, defaultVal string, profile *ProfileConfig, profileField func(*ProfileConfig) string) string {
	// 1. Explicit flag.
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetString(flagName)
		return val
	}

	// 2. Environment variable.
	if envVal := os.Getenv(envVar); envVal != "" {
		return envVal
	}

	// 3. Profile value.
	if profile != nil {
		if pv := profileField(profile); pv != "" {
			return pv
		}
	}

	// 4. Built-in default.
	return defaultVal
}

// resolveYes resolves the --yes / -y flag with precedence: flag > env.
func resolveYes(cmd *cobra.Command, _ *viper.Viper) bool {
	if cmd.Flags().Changed("yes") {
		val, _ := cmd.Flags().GetBool("yes")
		return val
	}

	envVal := os.Getenv("ICCTL_ASSUME_YES")
	switch strings.ToLower(envVal) {
	case "1", "true", "yes":
		return true
	}

	return false
}
