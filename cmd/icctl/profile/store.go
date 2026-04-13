package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// configFileName is the name of the profile configuration file.
const configFileName = "config.yaml"

// nameRegex validates profile names: lowercase alphanumeric, may contain
// hyphens, must start with an alphanumeric character.
var nameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ErrProfileNotFound indicates the named profile does not exist.
var ErrProfileNotFound = errors.New("profile: not found")

// ErrInvalidName indicates the profile name does not match the allowed pattern.
var ErrInvalidName = errors.New("profile: invalid name")

// Profile holds non-secret configuration for a named CLI profile.
// Secret material (tokens) is stored separately via SecretStore.
type Profile struct {
	Name            string `yaml:"-"`
	Environment     string `yaml:"environment,omitempty"`
	BaseURL         string `yaml:"base_url,omitempty"`
	DefaultOutput   string `yaml:"default_output,omitempty"`
	TracingEndpoint string `yaml:"tracing_endpoint,omitempty"`
}

// configFile represents the on-disk YAML structure.
type configFile struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

// Store manages profile configuration in a YAML file.
type Store struct {
	dir string
}

// NewStore creates a Store that persists to the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Save persists the profile. If a profile with the same name exists, it is
// overwritten. The name must match the pattern ^[a-z0-9][a-z0-9-]*$.
func (s *Store) Save(p Profile) error {
	if !nameRegex.MatchString(p.Name) {
		return fmt.Errorf("%w: %q does not match %s", ErrInvalidName, p.Name, nameRegex.String())
	}

	cfg, err := s.load()
	if err != nil {
		return err
	}

	cfg.Profiles[p.Name] = p
	return s.save(cfg)
}

// Get retrieves the profile with the given name.
func (s *Store) Get(name string) (Profile, error) {
	cfg, err := s.load()
	if err != nil {
		return Profile{}, err
	}

	p, ok := cfg.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}
	p.Name = name
	return p, nil
}

// List returns all profiles. The returned profiles never contain secrets.
func (s *Store) List() ([]Profile, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(cfg.Profiles))
	for name, p := range cfg.Profiles {
		p.Name = name
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// Remove deletes the named profile from the configuration file.
func (s *Store) Remove(name string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}

	delete(cfg.Profiles, name)

	// If the removed profile was the default, clear the default.
	if cfg.CurrentProfile == name {
		cfg.CurrentProfile = ""
	}
	return s.save(cfg)
}

// SetDefault sets the current_profile to the given name. The profile must
// already exist.
func (s *Store) SetDefault(name string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}

	cfg.CurrentProfile = name
	return s.save(cfg)
}

// Default returns the name of the current default profile. If no
// current_profile is set, it returns "default".
func (s *Store) Default() (string, error) {
	cfg, err := s.load()
	if err != nil {
		return "default", nil
	}
	if cfg.CurrentProfile == "" {
		return "default", nil
	}
	return cfg.CurrentProfile, nil
}

func (s *Store) configPath() string {
	return filepath.Join(s.dir, configFileName)
}

func (s *Store) load() (*configFile, error) {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &configFile{Profiles: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return &cfg, nil
}

func (s *Store) save(cfg *configFile) error {
	// Ensure parent directory exists with 0700.
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath(), data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
