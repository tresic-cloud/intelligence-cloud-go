package profile_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/profile"
)

// mockKeyring implements profile.KeyringBackend for testing.
type mockKeyring struct {
	store     map[string]string
	getErr    error
	setErr    error
	deleteErr error
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{store: make(map[string]string)}
}

func (m *mockKeyring) Get(service, account string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	v, ok := m.store[service+"/"+account]
	if !ok {
		return "", profile.ErrKeyringNotFound
	}
	return v, nil
}

func (m *mockKeyring) Set(service, account, password string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.store[service+"/"+account] = password
	return nil
}

func (m *mockKeyring) Delete(service, account string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.store, service+"/"+account)
	return nil
}

func TestSecretStore_KeychainSet_Get(t *testing.T) {
	kr := newMockKeyring()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(t.TempDir()),
	)

	cred := profile.Credential{
		AccessToken: "tok_abc",
		ExpiresAt:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := store.Set("staging", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get("staging")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q; want %q", got.AccessToken, cred.AccessToken)
	}
	if !got.ExpiresAt.Equal(cred.ExpiresAt) {
		t.Errorf("ExpiresAt = %v; want %v", got.ExpiresAt, cred.ExpiresAt)
	}

	// Verify keyring was used (JSON encoded).
	raw, err := kr.Get("icctl", "staging")
	if err != nil {
		t.Fatalf("keyring.Get: %v", err)
	}
	var stored struct {
		AccessToken string    `json:"access_token"`
		ExpiresAt   time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("Unmarshal keyring value: %v", err)
	}
	if stored.AccessToken != "tok_abc" {
		t.Errorf("keyring stored access_token = %q; want %q", stored.AccessToken, "tok_abc")
	}
}

func TestSecretStore_KeychainNotFound_FallsBackToFile(t *testing.T) {
	kr := newMockKeyring()
	kr.getErr = profile.ErrKeyringNotFound
	kr.setErr = profile.ErrKeyringNotFound

	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_fallback",
		ExpiresAt:   time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Set("default", cred); err != nil {
		t.Fatalf("Set (file fallback): %v", err)
	}

	// Reset getErr so we can test file fallback on Get too.
	kr.getErr = profile.ErrKeyringNotFound

	got, err := store.Get("default")
	if err != nil {
		t.Fatalf("Get (file fallback): %v", err)
	}
	if got.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q; want %q", got.AccessToken, cred.AccessToken)
	}

	// Verify file was created with 0600 perms.
	credFile := filepath.Join(dir, "credentials.yaml")
	info, err := os.Stat(credFile)
	if err != nil {
		t.Fatalf("Stat credentials.yaml: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o; want 0600", perm)
	}
}

func TestSecretStore_UnsupportedPlatform_FallsBackToFile(t *testing.T) {
	kr := newMockKeyring()
	kr.getErr = profile.ErrKeyringUnsupported
	kr.setErr = profile.ErrKeyringUnsupported

	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_unsupported",
		ExpiresAt:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := store.Set("prod", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get("prod")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q; want %q", got.AccessToken, cred.AccessToken)
	}
}

func TestSecretStore_DBusError_FallsBackToFile(t *testing.T) {
	dbusErr := errors.New("failed to connect to D-Bus session bus")
	kr := newMockKeyring()
	kr.getErr = dbusErr
	kr.setErr = dbusErr

	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_dbus",
		ExpiresAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := store.Set("myprof", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get("myprof")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q; want %q", got.AccessToken, cred.AccessToken)
	}
}

func TestSecretStore_Delete_RemovesBothKeychainAndFile(t *testing.T) {
	kr := newMockKeyring()
	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_delete",
		ExpiresAt:   time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}

	// Set in keychain.
	if err := store.Set("todelete", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Also write to file backend directly by making keychain fail temporarily.
	kr.setErr = profile.ErrKeyringNotFound
	if err := store.Set("todelete", cred); err != nil {
		t.Fatalf("Set (file): %v", err)
	}
	kr.setErr = nil

	if err := store.Delete("todelete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Keychain should be empty.
	_, err := kr.Get("icctl", "todelete")
	if !errors.Is(err, profile.ErrKeyringNotFound) {
		t.Errorf("keyring after delete: got err=%v, want ErrKeyringNotFound", err)
	}

	// File should not have the profile.
	_, err = store.Get("todelete")
	if err == nil {
		t.Error("Get after Delete: expected error, got nil")
	}
}

func TestSecretStore_Delete_KeychainFailIsWarningNotFatal(t *testing.T) {
	kr := newMockKeyring()
	kr.deleteErr = errors.New("keychain delete failed")

	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	// First set via file fallback (keychain set also fails).
	kr.setErr = profile.ErrKeyringNotFound
	cred := profile.Credential{
		AccessToken: "tok_warn",
		ExpiresAt:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := store.Set("warnprof", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}
	kr.setErr = nil

	// Delete should NOT return error even though keychain delete fails.
	if err := store.Delete("warnprof"); err != nil {
		t.Fatalf("Delete should not fail when keychain delete fails: %v", err)
	}
}

func TestSecretStore_Get_NotFound(t *testing.T) {
	kr := newMockKeyring()
	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("Get nonexistent: expected error, got nil")
	}
}

func TestSecretStore_CredentialJSON(t *testing.T) {
	// Verify the credential JSON encoding matches the spec.
	cred := profile.Credential{
		AccessToken: "eyJ...",
		ExpiresAt:   time.Date(2026, 4, 13, 20, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := m["access_token"]; !ok {
		t.Error("JSON missing access_token key")
	}
	if _, ok := m["expires_at"]; !ok {
		t.Error("JSON missing expires_at key")
	}
}

func TestSecretStore_FileCredentials_0600(t *testing.T) {
	kr := newMockKeyring()
	kr.setErr = profile.ErrKeyringUnsupported
	kr.getErr = profile.ErrKeyringUnsupported

	dir := t.TempDir()
	store := profile.NewSecretStore(
		profile.WithKeyring(kr),
		profile.WithConfigDir(dir),
	)

	cred := profile.Credential{
		AccessToken: "tok_perm",
		ExpiresAt:   time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := store.Set("permprof", cred); err != nil {
		t.Fatalf("Set: %v", err)
	}

	credFile := filepath.Join(dir, "credentials.yaml")
	info, err := os.Stat(credFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o; want 0600", perm)
	}
}
