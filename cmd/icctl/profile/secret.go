// Package profile provides CLI profile management including secret storage
// (keychain-first with file fallback) and profile configuration persistence.
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Credential holds the secret material associated with a profile.
type Credential struct {
	AccessToken string    `json:"access_token" yaml:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"   yaml:"expires_at"`
}

// Sentinel errors for keyring operations.
var (
	// ErrKeyringNotFound indicates the requested keyring entry does not exist.
	ErrKeyringNotFound = errors.New("keyring: entry not found")
	// ErrKeyringUnsupported indicates the platform has no keyring support.
	ErrKeyringUnsupported = errors.New("keyring: unsupported platform")
	// ErrCredentialNotFound indicates no credential exists for the profile.
	ErrCredentialNotFound = errors.New("profile: credential not found")
)

// keychainService is the service name used for all keyring operations.
const keychainService = "icctl"

// KeyringBackend abstracts the OS keyring so tests can provide a mock.
type KeyringBackend interface {
	Get(service, account string) (string, error)
	Set(service, account, password string) error
	Delete(service, account string) error
}

// SecretStore manages credential storage using a keychain-first strategy
// with a file-based fallback when the keychain is unavailable.
type SecretStore struct {
	keyring   KeyringBackend
	configDir string
}

// SecretStoreOption configures a SecretStore.
type SecretStoreOption func(*SecretStore)

// WithKeyring sets a custom keyring backend (primarily for testing).
func WithKeyring(kr KeyringBackend) SecretStoreOption {
	return func(s *SecretStore) { s.keyring = kr }
}

// WithConfigDir sets the configuration directory for file fallback.
func WithConfigDir(dir string) SecretStoreOption {
	return func(s *SecretStore) { s.configDir = dir }
}

// NewSecretStore creates a SecretStore with the given options.
func NewSecretStore(opts ...SecretStoreOption) *SecretStore {
	s := &SecretStore{}
	for _, opt := range opts {
		opt(s)
	}
	if s.keyring == nil {
		s.keyring = &realKeyring{}
	}
	if s.configDir == "" {
		s.configDir = defaultConfigDir()
	}
	return s
}

// Get retrieves the credential for the given profile. It tries the keychain
// first, falling back to the credentials file if the keychain is unavailable.
func (s *SecretStore) Get(profileName string) (Credential, error) {
	raw, err := s.keyring.Get(keychainService, profileName)
	if err == nil {
		return decodeCredential(raw)
	}
	if !isFallbackError(err) {
		return Credential{}, fmt.Errorf("keyring get %q: %w", profileName, err)
	}

	// Keychain unavailable — try file fallback.
	fb := newFileBackend(s.configDir)
	cred, err := fb.Get(profileName)
	if err != nil {
		return Credential{}, fmt.Errorf("credential not found for profile %q: %w", profileName, ErrCredentialNotFound)
	}
	return cred, nil
}

// Set stores the credential for the given profile. It writes to the keychain;
// if the keychain is unavailable, it falls back to the credentials file.
func (s *SecretStore) Set(profileName string, c Credential) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}

	err = s.keyring.Set(keychainService, profileName, string(data))
	if err == nil {
		return nil
	}
	if !isFallbackError(err) {
		return fmt.Errorf("keyring set %q: %w", profileName, err)
	}

	// Keychain unavailable — use file fallback.
	fb := newFileBackend(s.configDir)
	return fb.Set(profileName, c)
}

// Delete removes the credential for the given profile from both the keychain
// and the file fallback. Keychain delete failure is treated as a warning
// (non-fatal) per data-model.md section 2.1.
func (s *SecretStore) Delete(profileName string) error {
	// Try keychain delete — failure is a warning, not fatal.
	_ = s.keyring.Delete(keychainService, profileName)

	// Always try to remove from file fallback too.
	fb := newFileBackend(s.configDir)
	return fb.Delete(profileName)
}

// isFallbackError returns true if the error should trigger a file fallback
// rather than being treated as a hard failure.
func isFallbackError(err error) bool {
	if errors.Is(err, ErrKeyringNotFound) {
		return true
	}
	if errors.Is(err, ErrKeyringUnsupported) {
		return true
	}
	// D-Bus connection errors on Linux.
	if strings.Contains(err.Error(), "D-Bus") || strings.Contains(err.Error(), "dbus") {
		return true
	}
	return false
}

// decodeCredential parses a JSON-encoded credential string.
func decodeCredential(raw string) (Credential, error) {
	var c Credential
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return Credential{}, fmt.Errorf("decode credential: %w", err)
	}
	return c, nil
}
