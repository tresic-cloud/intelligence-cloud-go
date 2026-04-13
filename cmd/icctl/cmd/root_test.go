package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Mock implementations for testing
// ---------------------------------------------------------------------------

// mockProfileLoader implements ProfileLoader for tests.
type mockProfileLoader struct {
	profiles       map[string]*ProfileConfig
	currentProfile string
	loadErr        error
}

func (m *mockProfileLoader) Load(name string) (*ProfileConfig, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	if p, ok := m.profiles[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("profile %q not found", name)
}

func (m *mockProfileLoader) CurrentProfile() (string, error) {
	return m.currentProfile, nil
}

// Also satisfy ProfileStore for profile subcommand tests.
func (m *mockProfileLoader) List() ([]ProfileConfig, error) {
	var result []ProfileConfig
	for _, p := range m.profiles {
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockProfileLoader) Save(cfg *ProfileConfig) error {
	if m.profiles == nil {
		m.profiles = make(map[string]*ProfileConfig)
	}
	m.profiles[cfg.Name] = cfg
	return nil
}

func (m *mockProfileLoader) SetDefault(name string) error {
	if _, ok := m.profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	m.currentProfile = name
	return nil
}

func (m *mockProfileLoader) Remove(name string) error {
	if _, ok := m.profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(m.profiles, name)
	return nil
}

// mockSecretLoader implements SecretLoader for tests.
type mockSecretLoader struct {
	tokens    map[string]string
	loadErr   error
	saveErr   error
	removeErr error
}

func (m *mockSecretLoader) LoadToken(name string) (string, error) {
	if m.loadErr != nil {
		return "", m.loadErr
	}
	return m.tokens[name], nil
}

func (m *mockSecretLoader) SaveToken(name, token string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.tokens == nil {
		m.tokens = make(map[string]string)
	}
	m.tokens[name] = token
	return nil
}

func (m *mockSecretLoader) RemoveToken(name string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.tokens, name)
	return nil
}

// ---------------------------------------------------------------------------
// Tests: Global flag registration
// ---------------------------------------------------------------------------

func TestRootCmd_AllGlobalFlagsRegistered(t *testing.T) {
	cmd := NewRootCmd(nil, nil)

	expectedFlags := []struct {
		name      string
		shorthand string
	}{
		{"profile", ""},
		{"environment", ""},
		{"base-url", ""},
		{"token", ""},
		{"output", ""},
		{"yes", "y"},
		{"log-level", ""},
		{"request-id", ""},
		{"timeout", ""},
	}

	for _, f := range expectedFlags {
		t.Run(f.name, func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("expected global flag --%s to be registered", f.name)
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("expected shorthand %q for --%s, got %q", f.shorthand, f.name, flag.Shorthand)
			}
		})
	}
}

func TestRootCmd_FlagDefaults(t *testing.T) {
	cmd := NewRootCmd(nil, nil)

	tests := []struct {
		flag     string
		expected string
	}{
		{"profile", "default"},
		{"environment", ""},
		{"base-url", ""},
		{"token", ""},
		{"output", "table"},
		{"log-level", "warn"},
		{"request-id", ""},
		{"timeout", "30s"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("flag --%s not found", tt.flag)
			}
			if f.DefValue != tt.expected {
				t.Errorf("expected default %q for --%s, got %q", tt.expected, tt.flag, f.DefValue)
			}
		})
	}
}

func TestRootCmd_YesDefaultIsFalse(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	f := cmd.PersistentFlags().Lookup("yes")
	if f == nil {
		t.Fatal("flag --yes not found")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default false for --yes, got %q", f.DefValue)
	}
}

// ---------------------------------------------------------------------------
// Tests: 4-layer precedence (flag > env > profile > default)
// ---------------------------------------------------------------------------

func TestRootCmd_Precedence_Default(t *testing.T) {
	// With no flags, env, or profile: defaults apply.
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Add a dummy subcommand to exercise PersistentPreRunE.
	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState == nil {
		t.Fatal("state was not set in context")
	}
	if capturedState.ProfileName != "default" {
		t.Errorf("expected profile name %q, got %q", "default", capturedState.ProfileName)
	}
	if capturedState.EffectiveOutput != "table" {
		t.Errorf("expected output %q, got %q", "table", capturedState.EffectiveOutput)
	}
	if capturedState.LogLevel != "warn" {
		t.Errorf("expected log level %q, got %q", "warn", capturedState.LogLevel)
	}
	if capturedState.Timeout.String() != "30s" {
		t.Errorf("expected timeout 30s, got %s", capturedState.Timeout)
	}
}

func TestRootCmd_Precedence_FlagOverridesEnv(t *testing.T) {
	t.Setenv("ICCTL_OUTPUT", "json")

	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--output", "table", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveOutput != "table" {
		t.Errorf("expected flag to override env: got output %q", capturedState.EffectiveOutput)
	}
}

func TestRootCmd_Precedence_EnvOverridesProfile(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:          "default",
				Environment:   "staging",
				DefaultOutput: "json",
			},
		},
	}

	t.Setenv("ICCTL_OUTPUT", "table")

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveOutput != "table" {
		t.Errorf("expected env to override profile: got output %q", capturedState.EffectiveOutput)
	}
}

func TestRootCmd_Precedence_ProfileOverridesDefault(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:          "default",
				Environment:   "staging",
				DefaultOutput: "json",
				BaseURL:       "https://api.staging.intelligence.cloud",
			},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveOutput != "json" {
		t.Errorf("expected profile to override default: got output %q", capturedState.EffectiveOutput)
	}
	if capturedState.EffectiveEnvironment != "staging" {
		t.Errorf("expected profile env %q, got %q", "staging", capturedState.EffectiveEnvironment)
	}
	if capturedState.EffectiveBaseURL != "https://api.staging.intelligence.cloud" {
		t.Errorf("expected profile base URL, got %q", capturedState.EffectiveBaseURL)
	}
}

func TestRootCmd_Precedence_FlagOverridesAll(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:          "default",
				Environment:   "staging",
				DefaultOutput: "json",
			},
		},
	}

	t.Setenv("ICCTL_OUTPUT", "json")
	t.Setenv("ICCTL_LOG_LEVEL", "debug")

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--output", "table", "--log-level", "error", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveOutput != "table" {
		t.Errorf("expected flag output to override all: got %q", capturedState.EffectiveOutput)
	}
	if capturedState.LogLevel != "error" {
		t.Errorf("expected flag log-level to override all: got %q", capturedState.LogLevel)
	}
}

// ---------------------------------------------------------------------------
// Tests: Environment variable resolution
// ---------------------------------------------------------------------------

func TestRootCmd_EnvVarResolution(t *testing.T) {
	tests := []struct {
		envVar string
		value  string
		check  func(*State) (string, string)
	}{
		{
			envVar: "ICCTL_PROFILE",
			value:  "staging",
			check:  func(s *State) (string, string) { return s.ProfileName, "staging" },
		},
		{
			envVar: "ICCTL_TOKEN",
			value:  "tok_abc",
			check:  func(s *State) (string, string) { return s.EffectiveToken, "tok_abc" },
		},
		{
			envVar: "ICCTL_OUTPUT",
			value:  "json",
			check:  func(s *State) (string, string) { return s.EffectiveOutput, "json" },
		},
		{
			envVar: "ICCTL_LOG_LEVEL",
			value:  "debug",
			check:  func(s *State) (string, string) { return s.LogLevel, "debug" },
		},
		{
			envVar: "ICCTL_REQUEST_ID",
			value:  "req_123",
			check:  func(s *State) (string, string) { return s.RequestID, "req_123" },
		},
		{
			envVar: "ICCTL_TIMEOUT",
			value:  "10s",
			check:  func(s *State) (string, string) { return s.Timeout.String(), "10s" },
		},
		{
			envVar: "ICCTL_BASE_URL",
			value:  "https://custom.example.com",
			check:  func(s *State) (string, string) { return s.EffectiveBaseURL, "https://custom.example.com" },
		},
		{
			envVar: "ICCTL_ENVIRONMENT",
			value:  "production",
			check:  func(s *State) (string, string) { return s.EffectiveEnvironment, "production" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.envVar, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.value)

			cmd := NewRootCmd(nil, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			var capturedState *State
			sub := newTestSubcommand(func(s *State) { capturedState = s })
			cmd.AddCommand(sub)

			cmd.SetArgs([]string{"_test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, expected := tt.check(capturedState)
			if got != expected {
				t.Errorf("expected %q for %s, got %q", expected, tt.envVar, got)
			}
		})
	}
}

func TestRootCmd_AssumeYes_EnvVar(t *testing.T) {
	tests := []struct {
		envVal   string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"TRUE", true},
		{"YES", true},
		{"True", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("ICCTL_ASSUME_YES=%s", tt.envVal), func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("ICCTL_ASSUME_YES", tt.envVal)
			}

			cmd := NewRootCmd(nil, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			var capturedState *State
			sub := newTestSubcommand(func(s *State) { capturedState = s })
			cmd.AddCommand(sub)

			cmd.SetArgs([]string{"_test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedState.AssumeYes != tt.expected {
				t.Errorf("expected AssumeYes=%v for env=%q, got %v", tt.expected, tt.envVal, capturedState.AssumeYes)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: Flag override via command line
// ---------------------------------------------------------------------------

func TestRootCmd_FlagOverride_Token(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--token", "my-secret-token", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveToken != "my-secret-token" {
		t.Errorf("expected token %q, got %q", "my-secret-token", capturedState.EffectiveToken)
	}
}

func TestRootCmd_FlagOverride_BaseURL(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--base-url", "https://custom.api.example.com", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveBaseURL != "https://custom.api.example.com" {
		t.Errorf("expected base URL %q, got %q", "https://custom.api.example.com", capturedState.EffectiveBaseURL)
	}
}

func TestRootCmd_FlagOverride_Profile(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {
				Name:        "staging",
				Environment: "staging",
			},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--profile", "staging", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.ProfileName != "staging" {
		t.Errorf("expected profile %q, got %q", "staging", capturedState.ProfileName)
	}
}

func TestRootCmd_FlagOverride_Yes(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--yes", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedState.AssumeYes {
		t.Error("expected AssumeYes to be true")
	}
}

func TestRootCmd_FlagOverride_ShortYes(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"-y", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedState.AssumeYes {
		t.Error("expected AssumeYes to be true with -y")
	}
}

func TestRootCmd_FlagOverride_Timeout(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--timeout", "5m", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.Timeout.String() != "5m0s" {
		t.Errorf("expected timeout 5m0s, got %s", capturedState.Timeout)
	}
}

// ---------------------------------------------------------------------------
// Tests: Validation errors
// ---------------------------------------------------------------------------

func TestRootCmd_InvalidOutput(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	sub := newTestSubcommand(nil)
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--output", "xml", "_test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestRootCmd_InvalidLogLevel(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	sub := newTestSubcommand(nil)
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--log-level", "trace", "_test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestRootCmd_InvalidTimeout(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	sub := newTestSubcommand(nil)
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--timeout", "not-a-duration", "_test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestRootCmd_ExplicitProfileNotFound(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	sub := newTestSubcommand(nil)
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--profile", "nonexistent", "_test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent explicit profile")
	}
}

// ---------------------------------------------------------------------------
// Tests: Environment resolution
// ---------------------------------------------------------------------------

func TestRootCmd_EnvironmentResolvesBaseURL(t *testing.T) {
	tests := []struct {
		env     string
		baseURL string
	}{
		{"production", "https://api.intelligence.cloud"},
		{"staging", "https://api.staging.intelligence.cloud"},
		{"dev", "https://api.dev.intelligence.cloud"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cmd := NewRootCmd(nil, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			var capturedState *State
			sub := newTestSubcommand(func(s *State) { capturedState = s })
			cmd.AddCommand(sub)

			cmd.SetArgs([]string{"--environment", tt.env, "_test"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedState.EffectiveBaseURL != tt.baseURL {
				t.Errorf("expected base URL %q for env %q, got %q", tt.baseURL, tt.env, capturedState.EffectiveBaseURL)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: Token from secret loader
// ---------------------------------------------------------------------------

func TestRootCmd_TokenFromSecretLoader(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:        "default",
				Environment: "production",
			},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"default": "stored-secret-token",
		},
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveToken != "stored-secret-token" {
		t.Errorf("expected stored token, got %q", capturedState.EffectiveToken)
	}
}

func TestRootCmd_TokenFlagOverridesSecretLoader(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:        "default",
				Environment: "production",
			},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"default": "stored-secret-token",
		},
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"--token", "flag-token", "_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.EffectiveToken != "flag-token" {
		t.Errorf("expected flag token, got %q", capturedState.EffectiveToken)
	}
}

// ---------------------------------------------------------------------------
// Tests: State context propagation
// ---------------------------------------------------------------------------

func TestStateContext_RoundTrip(t *testing.T) {
	ctx := context.Background()
	state := &State{ProfileName: "test", EffectiveOutput: "json"}

	ctx = WithState(ctx, state)
	got := StateFrom(ctx)

	if got == nil {
		t.Fatal("expected state from context")
	}
	if got.ProfileName != "test" {
		t.Errorf("expected profile name %q, got %q", "test", got.ProfileName)
	}
}

func TestStateContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	got := StateFrom(ctx)
	if got != nil {
		t.Error("expected nil state from empty context")
	}
}

// ---------------------------------------------------------------------------
// Tests: Unknown subcommand / flag
// ---------------------------------------------------------------------------

func TestRootCmd_UnknownSubcommand(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"nonexistent-command"})
	err := cmd.Execute()
	// Cobra returns an error for unknown subcommands.
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestRootCmd_UnknownFlag(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--nonexistent-flag"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// ---------------------------------------------------------------------------
// Tests: CurrentProfile from config
// ---------------------------------------------------------------------------

func TestRootCmd_CurrentProfileFromConfig(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default", Environment: "production"},
			"staging": {Name: "staging", Environment: "staging"},
		},
		currentProfile: "staging",
	}

	// Ensure ICCTL_PROFILE is not set.
	os.Unsetenv("ICCTL_PROFILE")

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var capturedState *State
	sub := newTestSubcommand(func(s *State) { capturedState = s })
	cmd.AddCommand(sub)

	cmd.SetArgs([]string{"_test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState.ProfileName != "staging" {
		t.Errorf("expected current profile %q from config, got %q", "staging", capturedState.ProfileName)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestSubcommand creates a simple cobra command that captures state.
func newTestSubcommand(onRun func(*State)) *cobra.Command {
	return &cobra.Command{
		Use:    "_test",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if onRun != nil {
				state := StateFrom(cmd.Context())
				onRun(state)
			}
			return nil
		},
	}
}
