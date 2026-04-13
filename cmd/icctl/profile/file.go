package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// credentialsFileName is the name of the fallback credentials file.
const credentialsFileName = "credentials.yaml"

// fileBackend implements file-based credential storage as a fallback when the
// OS keychain is unavailable. Credentials are stored in YAML at
// $XDG_CONFIG_HOME/icctl/credentials.yaml with 0600 permissions.
type fileBackend struct {
	dir string
}

func newFileBackend(dir string) *fileBackend {
	return &fileBackend{dir: dir}
}

// credentialsFile is a map of profile name to Credential.
type credentialsFile map[string]Credential

func (fb *fileBackend) path() string {
	return filepath.Join(fb.dir, credentialsFileName)
}

// Get retrieves the credential for the named profile from the file.
func (fb *fileBackend) Get(profileName string) (Credential, error) {
	creds, err := fb.load()
	if err != nil {
		return Credential{}, err
	}
	c, ok := creds[profileName]
	if !ok {
		return Credential{}, ErrCredentialNotFound
	}
	return c, nil
}

// Set writes the credential for the named profile to the file.
func (fb *fileBackend) Set(profileName string, c Credential) error {
	creds, err := fb.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if creds == nil {
		creds = make(credentialsFile)
	}
	creds[profileName] = c
	return fb.save(creds)
}

// Delete removes the credential for the named profile from the file.
func (fb *fileBackend) Delete(profileName string) error {
	creds, err := fb.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	delete(creds, profileName)
	return fb.save(creds)
}

func (fb *fileBackend) load() (credentialsFile, error) {
	data, err := os.ReadFile(fb.path())
	if err != nil {
		return nil, err
	}
	var creds credentialsFile
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse %s: %w", fb.path(), err)
	}
	return creds, nil
}

func (fb *fileBackend) save(creds credentialsFile) error {
	// Ensure parent directory exists with 0700.
	if err := os.MkdirAll(fb.dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(fb.path(), data, 0600); err != nil {
		return fmt.Errorf("write %s: %w", fb.path(), err)
	}
	return nil
}

// defaultConfigDir returns the default configuration directory following
// XDG Base Directory specification.
func defaultConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "icctl")
	}
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "icctl")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "icctl")
}
