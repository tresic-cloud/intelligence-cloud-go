package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// ProfileStore is the full interface for managing profile configuration.
// W3-1's profile.Store will satisfy this interface at merge.
type ProfileStore interface {
	ProfileLoader

	// List returns all profile names.
	List() ([]ProfileConfig, error)

	// Save persists a profile configuration.
	Save(cfg *ProfileConfig) error

	// SetDefault sets the current_profile in config.yaml.
	SetDefault(name string) error

	// Remove deletes a profile from config.yaml.
	Remove(name string) error
}

// SecretStore is the full interface for managing profile secrets.
// W3-1's profile.SecretStore will satisfy this interface at merge.
type SecretStore interface {
	SecretLoader

	// SaveToken stores a bearer token for the given profile.
	SaveToken(name, token string) error

	// RemoveToken deletes the stored token for the given profile.
	RemoveToken(name string) error
}

// ClientTester is an interface for testing profile credentials by calling
// the API. This avoids a direct dependency on the tui package from the
// cmd package.
type ClientTester interface {
	// TestCredentials validates that the given base URL and token are
	// accepted by the API. Returns nil on success.
	TestCredentials(baseURL, token string) error
}

// validProfileName matches the profile name constraint: [a-z0-9][a-z0-9-]*
var validProfileName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// NewProfileCmd creates the profile subcommand tree with all sub-subcommands.
func NewProfileCmd(profileLoader ProfileLoader, secretLoader SecretLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage CLI profiles",
		Long:  "Manage named CLI configuration profiles including environments, base URLs, and credentials.",
		Example: `  # List all profiles
  icctl profile list

  # Show a specific profile
  icctl profile show staging

  # Add a new profile
  icctl profile add staging --environment staging --token-stdin

  # Set the default profile
  icctl profile set-default staging`,
	}

	// Sub-subcommands delegate to package-level constructors.
	cmd.AddCommand(newProfileListCmd(profileLoader))
	cmd.AddCommand(newProfileShowCmd(profileLoader))
	cmd.AddCommand(newProfileAddCmd(profileLoader, secretLoader))
	cmd.AddCommand(newProfileSetDefaultCmd(profileLoader))
	cmd.AddCommand(newProfileSetTokenCmd(secretLoader))
	cmd.AddCommand(newProfileRemoveCmd(profileLoader, secretLoader))
	cmd.AddCommand(newProfileTestCmd(profileLoader, secretLoader))

	return cmd
}

// ---------------------------------------------------------------------------
// profile list
// ---------------------------------------------------------------------------

func newProfileListCmd(loader ProfileLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Long:  "List all configured profiles. Tokens are never shown in the output.",
		Example: `  icctl profile list
  icctl profile list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileList(cmd, loader)
		},
	}
}

func runProfileList(cmd *cobra.Command, loader ProfileLoader) error {
	store, ok := loader.(ProfileStore)
	if !ok || store == nil {
		return fmt.Errorf("profile store not available")
	}

	profiles, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	outputFmt := "table"
	if state := StateFrom(cmd.Context()); state != nil {
		outputFmt = state.EffectiveOutput
	}

	currentProfile := ""
	if state := StateFrom(cmd.Context()); state != nil {
		currentProfile = state.ProfileName
	}

	switch outputFmt {
	case "json":
		type jsonProfile struct {
			Name        string `json:"name"`
			Environment string `json:"environment,omitempty"`
			BaseURL     string `json:"base_url,omitempty"`
			Default     bool   `json:"default"`
		}
		items := make([]jsonProfile, 0, len(profiles))
		for _, p := range profiles {
			items = append(items, jsonProfile{
				Name:        p.Name,
				Environment: p.Environment,
				BaseURL:     p.BaseURL,
				Default:     p.Name == currentProfile,
			})
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{"items": items})
	default:
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tENVIRONMENT\tBASE URL\tDEFAULT")
		for _, p := range profiles {
			isDefault := ""
			if p.Name == currentProfile {
				isDefault = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Environment, p.BaseURL, isDefault)
		}
		return w.Flush()
	}
}

// ---------------------------------------------------------------------------
// profile show
// ---------------------------------------------------------------------------

func newProfileShowCmd(loader ProfileLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show profile details",
		Long:  "Show the non-secret configuration of a profile. Tokens are never shown.",
		Example: `  icctl profile show staging
  icctl profile show staging --output json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			} else if state := StateFrom(cmd.Context()); state != nil {
				name = state.ProfileName
			}
			return runProfileShow(cmd, loader, name)
		},
	}
}

func runProfileShow(cmd *cobra.Command, loader ProfileLoader, name string) error {
	if loader == nil {
		return fmt.Errorf("profile store not available")
	}

	cfg, err := loader.Load(name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	outputFmt := "table"
	if state := StateFrom(cmd.Context()); state != nil {
		outputFmt = state.EffectiveOutput
	}

	switch outputFmt {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"name":           cfg.Name,
			"environment":    cfg.Environment,
			"base_url":       cfg.BaseURL,
			"default_output": cfg.DefaultOutput,
		})
	default:
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Name:           %s\n", cfg.Name)
		fmt.Fprintf(w, "Environment:    %s\n", cfg.Environment)
		fmt.Fprintf(w, "Base URL:       %s\n", cfg.BaseURL)
		fmt.Fprintf(w, "Default output: %s\n", cfg.DefaultOutput)
		return nil
	}
}

// ---------------------------------------------------------------------------
// profile add
// ---------------------------------------------------------------------------

func newProfileAddCmd(loader ProfileLoader, secrets SecretLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new profile",
		Long:  "Add a new named profile with an environment, base URL, and optional token read from stdin.",
		Example: `  # Add with explicit settings
  icctl profile add staging --environment staging

  # Add with token from stdin
  echo "tok_abc123" | icctl profile add staging --environment staging --token-stdin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileAdd(cmd, loader, secrets, args[0])
		},
	}
	cmd.Flags().Bool("token-stdin", false, "Read token from stdin")
	return cmd
}

func runProfileAdd(cmd *cobra.Command, loader ProfileLoader, secrets SecretLoader, name string) error {
	if !validProfileName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must match [a-z0-9][a-z0-9-]*", name)
	}

	store, ok := loader.(ProfileStore)
	if !ok || store == nil {
		return fmt.Errorf("profile store not available")
	}

	// Read parent flags for environment and base-url.
	environment, _ := cmd.Flags().GetString("environment")
	baseURL, _ := cmd.Flags().GetString("base-url")

	cfg := &ProfileConfig{
		Name:        name,
		Environment: environment,
		BaseURL:     baseURL,
	}

	if err := store.Save(cfg); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// Handle --token-stdin.
	tokenStdin, _ := cmd.Flags().GetBool("token-stdin")
	if tokenStdin {
		secretStore, ok := secrets.(SecretStore)
		if !ok || secretStore == nil {
			return fmt.Errorf("secret store not available")
		}

		token, err := readTokenFromStdin(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read token from stdin: %w", err)
		}

		if err := secretStore.SaveToken(name, token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Profile %q added.\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// profile set-default
// ---------------------------------------------------------------------------

func newProfileSetDefaultCmd(loader ProfileLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default profile",
		Long:  "Set which profile is used when --profile is not specified and ICCTL_PROFILE is not set.",
		Example: `  icctl profile set-default staging`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSetDefault(cmd, loader, args[0])
		},
	}
}

func runProfileSetDefault(cmd *cobra.Command, loader ProfileLoader, name string) error {
	store, ok := loader.(ProfileStore)
	if !ok || store == nil {
		return fmt.Errorf("profile store not available")
	}

	// Verify profile exists.
	if _, err := store.Load(name); err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	if err := store.SetDefault(name); err != nil {
		return fmt.Errorf("failed to set default profile: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Default profile set to %q.\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// profile set-token
// ---------------------------------------------------------------------------

func newProfileSetTokenCmd(secrets SecretLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-token <name>",
		Short: "Set the token for a profile",
		Long:  "Set or update the bearer token for a profile. The token is read from stdin when --token-stdin is used.",
		Example: `  echo "tok_abc123" | icctl profile set-token staging --token-stdin`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSetToken(cmd, secrets, args[0])
		},
	}
	cmd.Flags().Bool("token-stdin", false, "Read token from stdin")
	return cmd
}

func runProfileSetToken(cmd *cobra.Command, secrets SecretLoader, name string) error {
	secretStore, ok := secrets.(SecretStore)
	if !ok || secretStore == nil {
		return fmt.Errorf("secret store not available")
	}

	tokenStdin, _ := cmd.Flags().GetBool("token-stdin")
	if !tokenStdin {
		return fmt.Errorf("--token-stdin is required")
	}

	token, err := readTokenFromStdin(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("failed to read token from stdin: %w", err)
	}

	if err := secretStore.SaveToken(name, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Token updated for profile %q.\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// profile remove
// ---------------------------------------------------------------------------

func newProfileRemoveCmd(loader ProfileLoader, secrets SecretLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a profile",
		Long:  "Remove a profile and its associated token. Failure to delete the keychain entry is logged as a warning but does not fail the command.",
		Example: `  icctl profile remove staging`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileRemove(cmd, loader, secrets, args[0])
		},
	}
}

func runProfileRemove(cmd *cobra.Command, loader ProfileLoader, secrets SecretLoader, name string) error {
	store, ok := loader.(ProfileStore)
	if !ok || store == nil {
		return fmt.Errorf("profile store not available")
	}

	if err := store.Remove(name); err != nil {
		return fmt.Errorf("failed to remove profile: %w", err)
	}

	// Attempt to remove the token; failure is a warning, not an error.
	if secretStore, ok := secrets.(SecretStore); ok && secretStore != nil {
		if err := secretStore.RemoveToken(name); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to remove token for profile %q: %v\n", name, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Profile %q removed.\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// profile test
// ---------------------------------------------------------------------------

func newProfileTestCmd(loader ProfileLoader, secrets SecretLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "test <name>",
		Short: "Test profile credentials",
		Long:  "Test that a profile's credentials are valid by making an API call (Me.Get).",
		Example: `  icctl profile test staging`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileTest(cmd, loader, secrets, args[0])
		},
	}
}

func runProfileTest(cmd *cobra.Command, loader ProfileLoader, secrets SecretLoader, name string) error {
	if loader == nil {
		return fmt.Errorf("profile store not available")
	}

	cfg, err := loader.Load(name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	// Resolve the base URL.
	baseURL := cfg.BaseURL
	if baseURL == "" && cfg.Environment != "" {
		if u, ok := environmentBaseURLs[cfg.Environment]; ok {
			baseURL = u
		}
	}
	if baseURL == "" {
		return fmt.Errorf("profile %q has no base URL or known environment", name)
	}

	// Load the token.
	var token string
	if secrets != nil {
		token, err = secrets.LoadToken(name)
		if err != nil {
			return fmt.Errorf("failed to load token for profile %q: %w", name, err)
		}
	}
	if token == "" {
		return fmt.Errorf("no token available for profile %q", name)
	}

	// If a ClientTester is available in context (set by integration harness), use it.
	if tester := clientTesterFrom(cmd.Context()); tester != nil {
		if err := tester.TestCredentials(baseURL, token); err != nil {
			return fmt.Errorf("credential test failed: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Profile %q: credentials are valid.\n", name)
		return nil
	}

	// Without a tester, we cannot validate credentials. This path is used
	// when the real tui.BuildClient is wired in via main.go; during unit
	// tests a ClientTester mock is injected.
	fmt.Fprintf(cmd.OutOrStdout(), "Profile %q: credentials loaded (no API test available in this context).\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// readTokenFromStdin reads a single line (the token) from the given reader.
func readTokenFromStdin(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("empty input")
}

// clientTesterKey is the context key for ClientTester injection.
type clientTesterKey struct{}

// WithClientTester returns a context carrying a ClientTester for use by
// profile test.
func WithClientTester(ctx context.Context, t ClientTester) context.Context {
	return context.WithValue(ctx, clientTesterKey{}, t)
}

// clientTesterFrom extracts the ClientTester from the context.
func clientTesterFrom(ctx context.Context) ClientTester {
	t, _ := ctx.Value(clientTesterKey{}).(ClientTester)
	return t
}
