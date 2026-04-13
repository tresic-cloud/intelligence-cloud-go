package profile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/profile"
)

func TestStore_SaveAndGet_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{
		Name:            "staging",
		Environment:     "staging",
		BaseURL:         "https://api.staging.intelligence.cloud",
		DefaultOutput:   "json",
		TracingEndpoint: "https://otlp.internal.example.com",
	}

	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get("staging")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != p.Name {
		t.Errorf("Name = %q; want %q", got.Name, p.Name)
	}
	if got.Environment != p.Environment {
		t.Errorf("Environment = %q; want %q", got.Environment, p.Environment)
	}
	if got.BaseURL != p.BaseURL {
		t.Errorf("BaseURL = %q; want %q", got.BaseURL, p.BaseURL)
	}
	if got.DefaultOutput != p.DefaultOutput {
		t.Errorf("DefaultOutput = %q; want %q", got.DefaultOutput, p.DefaultOutput)
	}
	if got.TracingEndpoint != p.TracingEndpoint {
		t.Errorf("TracingEndpoint = %q; want %q", got.TracingEndpoint, p.TracingEndpoint)
	}
}

func TestStore_ConfigFile_0600(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{
		Name:        "default",
		Environment: "production",
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o; want 0600", perm)
	}
}

func TestStore_NameValidation(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"default", false},
		{"my-profile", false},
		{"staging1", false},
		{"a", false},
		{"123", false},
		{"a-b-c", false},
		{"", true},
		{"-invalid", true},
		{"UPPER", true},
		{"has space", true},
		{"under_score", true},
		{"has.dot", true},
		{"special!", true},
		{"-start", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := profile.Profile{
				Name:        tt.name,
				Environment: "production",
			}
			err := store.Save(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save(%q) error = %v; wantErr = %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestStore_SetDefault(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p1 := profile.Profile{Name: "alpha", Environment: "production"}
	p2 := profile.Profile{Name: "beta", Environment: "staging"}

	if err := store.Save(p1); err != nil {
		t.Fatalf("Save alpha: %v", err)
	}
	if err := store.Save(p2); err != nil {
		t.Fatalf("Save beta: %v", err)
	}

	if err := store.SetDefault("beta"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}

	name, err := store.Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if name != "beta" {
		t.Errorf("Default = %q; want %q", name, "beta")
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := store.Save(profile.Profile{Name: name, Environment: "prod"}); err != nil {
			t.Fatalf("Save %q: %v", name, err)
		}
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("List len = %d; want 3", len(profiles))
	}

	names := make(map[string]bool)
	for _, p := range profiles {
		names[p.Name] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("List missing profile %q", want)
		}
	}
}

func TestStore_Remove(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{Name: "toremove", Environment: "staging"}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Remove("toremove"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err := store.Get("toremove")
	if err == nil {
		t.Error("Get after Remove: expected error, got nil")
	}
}

func TestStore_Remove_NonExistent(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	err := store.Remove("nonexistent")
	if err == nil {
		t.Error("Remove nonexistent: expected error, got nil")
	}
}

func TestStore_Get_NonExistent(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("Get nonexistent: expected error, got nil")
	}
}

func TestStore_SetDefault_NonExistent(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	err := store.SetDefault("nonexistent")
	if err == nil {
		t.Error("SetDefault nonexistent: expected error, got nil")
	}
}

func TestStore_NoTokenInConfig(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{
		Name:        "secure",
		Environment: "production",
		BaseURL:     "https://api.intelligence.cloud",
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := strings.ToLower(string(data))
	for _, forbidden := range []string{"token", "secret", "password", "credential", "access_token"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("config.yaml contains forbidden term %q:\n%s", forbidden, string(data))
		}
	}
}

func TestStore_MultipleProfiles_Overwrite(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{Name: "evolving", Environment: "staging", BaseURL: "https://old.url"}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	p.Environment = "production"
	p.BaseURL = "https://new.url"
	if err := store.Save(p); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, err := store.Get("evolving")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Environment != "production" {
		t.Errorf("Environment = %q; want %q", got.Environment, "production")
	}
	if got.BaseURL != "https://new.url" {
		t.Errorf("BaseURL = %q; want %q", got.BaseURL, "https://new.url")
	}
}

func TestStore_Default_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	store := profile.NewStore(dir)

	name, err := store.Default()
	if err != nil {
		t.Fatalf("Default on empty store: %v", err)
	}
	// Default should return "default" when no current_profile is set.
	if name != "default" {
		t.Errorf("Default = %q; want %q", name, "default")
	}
}
