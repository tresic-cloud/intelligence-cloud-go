package profile_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/profile"
)

func TestFilePermissions_ConfigFile0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	dir := t.TempDir()
	store := profile.NewStore(dir)

	p := profile.Profile{
		Name:        "permtest",
		Environment: "staging",
		BaseURL:     "https://api.staging.intelligence.cloud",
	}

	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat config.yaml: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.yaml permissions = %04o; want 0600", perm)
	}
}

func TestFilePermissions_CredentialsFile0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	dir := t.TempDir()

	// Use a mock keyring that always fails, forcing file fallback.
	ss := profile.NewSecretStore(
		profile.WithKeyring(&failingKeyring{}),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_test_perm_audit",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	if err := ss.Set("permtest", cred); err != nil {
		t.Fatalf("SecretStore.Set: %v", err)
	}

	credsPath := filepath.Join(dir, "credentials.yaml")
	info, err := os.Stat(credsPath)
	if err != nil {
		t.Fatalf("Stat credentials.yaml: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials.yaml permissions = %04o; want 0600", perm)
	}
}

func TestFilePermissions_ParentDir0700(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	// Use a subdirectory under TempDir to ensure the Store creates it.
	base := t.TempDir()
	dir := filepath.Join(base, "icctl")

	store := profile.NewStore(dir)

	p := profile.Profile{
		Name:        "dirtest",
		Environment: "production",
	}

	if err := store.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat config dir: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("config dir permissions = %04o; want 0700", perm)
	}
}

func TestFilePermissions_SurviveOverwrite_Config(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	dir := t.TempDir()
	store := profile.NewStore(dir)

	// First write.
	p := profile.Profile{
		Name:        "overwrite",
		Environment: "staging",
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save (first): %v", err)
	}

	// Second write (overwrite).
	p.Environment = "production"
	if err := store.Save(p); err != nil {
		t.Fatalf("Save (second): %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat config.yaml after overwrite: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.yaml permissions after overwrite = %04o; want 0600", perm)
	}
}

func TestFilePermissions_SurviveOverwrite_Credentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	dir := t.TempDir()

	ss := profile.NewSecretStore(
		profile.WithKeyring(&failingKeyring{}),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_first",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	// First write.
	if err := ss.Set("overwrite", cred); err != nil {
		t.Fatalf("SecretStore.Set (first): %v", err)
	}

	// Second write (overwrite).
	cred.AccessToken = "tok_second"
	if err := ss.Set("overwrite", cred); err != nil {
		t.Fatalf("SecretStore.Set (second): %v", err)
	}

	credsPath := filepath.Join(dir, "credentials.yaml")
	info, err := os.Stat(credsPath)
	if err != nil {
		t.Fatalf("Stat credentials.yaml after overwrite: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials.yaml permissions after overwrite = %04o; want 0600", perm)
	}
}

func TestFilePermissions_ParentDir0700_CredentialsFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm model differs on Windows")
	}

	base := t.TempDir()
	dir := filepath.Join(base, "icctl")

	ss := profile.NewSecretStore(
		profile.WithKeyring(&failingKeyring{}),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_dirperm",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	if err := ss.Set("dirperm", cred); err != nil {
		t.Fatalf("SecretStore.Set: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat credentials dir: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("credentials dir permissions = %04o; want 0700", perm)
	}
}

// failingKeyring always returns ErrKeyringUnsupported, forcing the file
// fallback path in SecretStore.
type failingKeyring struct{}

func (f *failingKeyring) Get(_, _ string) (string, error) {
	return "", profile.ErrKeyringUnsupported
}

func (f *failingKeyring) Set(_, _, _ string) error {
	return profile.ErrKeyringUnsupported
}

func (f *failingKeyring) Delete(_, _ string) error {
	return profile.ErrKeyringUnsupported
}
