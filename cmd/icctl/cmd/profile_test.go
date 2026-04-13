package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests: profile list
// ---------------------------------------------------------------------------

func TestProfileList_TableOutput(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default":    {Name: "default", Environment: "production", BaseURL: "https://api.intelligence.cloud"},
			"staging":    {Name: "staging", Environment: "staging", BaseURL: "https://api.staging.intelligence.cloud"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected table header with NAME")
	}
	if !strings.Contains(output, "default") {
		t.Error("expected default profile in output")
	}
	if !strings.Contains(output, "staging") {
		t.Error("expected staging profile in output")
	}
	// Tokens must never appear in list output.
	if strings.Contains(output, "tok_") || strings.Contains(output, "TOKEN") {
		t.Error("token-related content should never appear in profile list")
	}
}

func TestProfileList_JSONOutput(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default", Environment: "production"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--output", "json", "profile", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, buf.String())
	}

	if _, ok := result["items"]; !ok {
		t.Error("expected 'items' key in JSON output")
	}

	// Verify no tokens in JSON.
	if strings.Contains(buf.String(), "token") {
		t.Error("JSON output should never contain token fields")
	}
}

func TestProfileList_NeverRevealsTokens(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default", Environment: "production"},
		},
	}

	for _, format := range []string{"table", "json"} {
		t.Run(format, func(t *testing.T) {
			cmd := NewRootCmd(loader, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			args := []string{"profile", "list"}
			if format == "json" {
				args = []string{"--output", "json", "profile", "list"}
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			if strings.Contains(strings.ToLower(output), "secret") ||
				strings.Contains(strings.ToLower(output), "bearer") {
				t.Error("profile list output must never reveal secrets")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: profile show
// ---------------------------------------------------------------------------

func TestProfileShow_TableOutput(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging", BaseURL: "https://api.staging.intelligence.cloud", DefaultOutput: "json"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "show", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "staging") {
		t.Error("expected profile name in output")
	}
	if !strings.Contains(output, "json") {
		t.Error("expected default output format in output")
	}
}

func TestProfileShow_JSONOutput(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--output", "json", "profile", "show", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["name"] != "staging" {
		t.Errorf("expected name %q, got %v", "staging", result["name"])
	}
}

func TestProfileShow_Nonexistent(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "show", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestProfileShow_NeverRevealsTokens(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "show", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(strings.ToLower(output), "token") ||
		strings.Contains(strings.ToLower(output), "secret") {
		t.Error("profile show must never reveal tokens or secrets")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile add
// ---------------------------------------------------------------------------

func TestProfileAdd_Success(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "add", "staging", "--environment", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := loader.profiles["staging"]; !ok {
		t.Error("expected profile 'staging' to be saved")
	}

	output := buf.String()
	if !strings.Contains(output, "added") {
		t.Error("expected confirmation message")
	}
}

func TestProfileAdd_WithTokenStdin(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{},
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Simulate stdin with a token.
	cmd.SetIn(strings.NewReader("tok_my_secret_123\n"))

	cmd.SetArgs([]string{"profile", "add", "staging", "--environment", "staging", "--token-stdin"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secrets.tokens["staging"] != "tok_my_secret_123" {
		t.Errorf("expected token to be saved, got %q", secrets.tokens["staging"])
	}
}

func TestProfileAdd_InvalidName(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	invalidNames := []string{
		"UPPERCASE",
		"has spaces",
		"-starts-with-dash",
		"has_underscore",
		"has.dot",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCmd(loader, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			cmd.SetArgs([]string{"profile", "add", name})
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for invalid profile name %q", name)
			}
		})
	}
}

func TestProfileAdd_ValidNames(t *testing.T) {
	validNames := []string{
		"default",
		"staging",
		"prod-us-east-1",
		"a",
		"123",
		"my-profile-2",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			loader := &mockProfileLoader{
				profiles: map[string]*ProfileConfig{},
			}
			cmd := NewRootCmd(loader, nil)
			cmd.SetContext(context.Background())
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			cmd.SetArgs([]string{"profile", "add", name})
			err := cmd.Execute()
			if err != nil {
				t.Errorf("unexpected error for valid profile name %q: %v", name, err)
			}
		})
	}
}

func TestProfileAdd_MissingName(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "add"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing profile name argument")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile set-default
// ---------------------------------------------------------------------------

func TestProfileSetDefault_Success(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default"},
			"staging": {Name: "staging"},
		},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "set-default", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loader.currentProfile != "staging" {
		t.Errorf("expected current profile %q, got %q", "staging", loader.currentProfile)
	}
}

func TestProfileSetDefault_Nonexistent(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "set-default", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile set-token
// ---------------------------------------------------------------------------

func TestProfileSetToken_Success(t *testing.T) {
	secrets := &mockSecretLoader{
		tokens: map[string]string{},
	}

	cmd := NewRootCmd(nil, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader("new-token-value\n"))

	cmd.SetArgs([]string{"profile", "set-token", "staging", "--token-stdin"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secrets.tokens["staging"] != "new-token-value" {
		t.Errorf("expected token %q, got %q", "new-token-value", secrets.tokens["staging"])
	}
}

func TestProfileSetToken_WithoutTokenStdin(t *testing.T) {
	secrets := &mockSecretLoader{
		tokens: map[string]string{},
	}

	cmd := NewRootCmd(nil, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "set-token", "staging"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --token-stdin is not set")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile remove
// ---------------------------------------------------------------------------

func TestProfileRemove_Success(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging"},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"staging": "tok_abc",
		},
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "remove", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := loader.profiles["staging"]; ok {
		t.Error("expected profile 'staging' to be removed")
	}
	if _, ok := secrets.tokens["staging"]; ok {
		t.Error("expected token for 'staging' to be removed")
	}
}

func TestProfileRemove_Nonexistent(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "remove", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for removing nonexistent profile")
	}
}

func TestProfileRemove_TokenDeleteFailureIsWarning(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging"},
		},
	}
	secrets := &mockSecretLoader{
		tokens:    map[string]string{},
		removeErr: fmt.Errorf("keychain unavailable"),
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	cmd.SetArgs([]string{"profile", "remove", "staging"})
	err := cmd.Execute()
	// The command should succeed even if token removal fails.
	if err != nil {
		t.Fatalf("expected no error (warning only), got: %v", err)
	}

	// The warning should appear on stderr.
	if !strings.Contains(errBuf.String(), "Warning") {
		t.Error("expected warning on stderr about token removal failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile test
// ---------------------------------------------------------------------------

func TestProfileTest_Success(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging"},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"staging": "tok_valid",
		},
	}

	// Create a mock client tester.
	tester := &mockClientTester{err: nil}

	cmd := NewRootCmd(loader, secrets)
	ctx := WithClientTester(context.Background(), tester)
	cmd.SetContext(ctx)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "test", "staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "valid") {
		t.Error("expected success message")
	}
}

func TestProfileTest_NoToken(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging"},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{},
	}

	cmd := NewRootCmd(loader, secrets)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "test", "staging"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no token available")
	}
}

func TestProfileTest_CredentialTestFails(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"staging": {Name: "staging", Environment: "staging"},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"staging": "tok_invalid",
		},
	}

	tester := &mockClientTester{err: fmt.Errorf("401 unauthorized")}

	cmd := NewRootCmd(loader, secrets)
	ctx := WithClientTester(context.Background(), tester)
	cmd.SetContext(ctx)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "test", "staging"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when credential test fails")
	}
}

func TestProfileTest_NonexistentProfile(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{},
	}

	cmd := NewRootCmd(loader, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"profile", "test", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

// ---------------------------------------------------------------------------
// Tests: profile subcommand tree
// ---------------------------------------------------------------------------

func TestProfile_SubcommandTree(t *testing.T) {
	cmd := NewRootCmd(nil, nil)

	profileCmd, _, err := cmd.Find([]string{"profile"})
	if err != nil {
		t.Fatalf("profile command not found: %v", err)
	}

	expectedSubcommands := []string{
		"list", "show", "add", "set-default", "set-token", "remove", "test",
	}

	for _, name := range expectedSubcommands {
		t.Run(name, func(t *testing.T) {
			found := false
			for _, sub := range profileCmd.Commands() {
				if sub.Name() == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected subcommand %q under profile", name)
			}
		})
	}
}

func TestProfile_AllSubcommandsHaveHelp(t *testing.T) {
	cmd := NewRootCmd(nil, nil)

	profileCmd, _, err := cmd.Find([]string{"profile"})
	if err != nil {
		t.Fatalf("profile command not found: %v", err)
	}

	for _, sub := range profileCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("subcommand %q missing Short description", sub.Name())
			}
			if sub.Long == "" {
				t.Errorf("subcommand %q missing Long description", sub.Name())
			}
			if sub.Example == "" {
				t.Errorf("subcommand %q missing Example", sub.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: profile name validation
// ---------------------------------------------------------------------------

func TestValidProfileName(t *testing.T) {
	valid := []string{"default", "staging", "prod-us", "a", "123", "a-b-c"}
	invalid := []string{"", "-start", "Upper", "has space", "under_score"}

	for _, name := range valid {
		if !validProfileName.MatchString(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	for _, name := range invalid {
		if validProfileName.MatchString(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Mock client tester
// ---------------------------------------------------------------------------

type mockClientTester struct {
	err error
}

func (m *mockClientTester) TestCredentials(baseURL, token string) error {
	return m.err
}
