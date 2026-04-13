package tui

import (
	"errors"
	"testing"
)

func TestBuildClient_ValidInputs(t *testing.T) {
	client, err := BuildClient(
		"https://api.intelligence.cloud",
		"tok_abc123",
		"warn",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBuildClient_EmptyToken(t *testing.T) {
	_, err := BuildClient(
		"https://api.intelligence.cloud",
		"",
		"warn",
	)
	if err == nil {
		t.Fatal("expected error for empty token")
	}

	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestBuildClient_EmptyBaseURL(t *testing.T) {
	_, err := BuildClient(
		"",
		"tok_abc123",
		"warn",
	)
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}

	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestBuildClient_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", ""}

	for _, level := range levels {
		t.Run("level="+level, func(t *testing.T) {
			client, err := BuildClient(
				"https://api.intelligence.cloud",
				"tok_abc123",
				level,
			)
			if err != nil {
				t.Fatalf("unexpected error for log level %q: %v", level, err)
			}
			if client == nil {
				t.Fatal("expected non-nil client")
			}
		})
	}
}

func TestBuildClient_UnknownLogLevelDefaultsToWarn(t *testing.T) {
	// Unknown log levels should not error; they default to warn.
	client, err := BuildClient(
		"https://api.intelligence.cloud",
		"tok_abc123",
		"unknown-level",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBuildClient_LocalhostHTTPAllowed(t *testing.T) {
	client, err := BuildClient(
		"http://localhost:8080",
		"tok_abc123",
		"warn",
	)
	if err != nil {
		t.Fatalf("unexpected error for localhost HTTP: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBuildClient_NonLocalhostHTTPRejected(t *testing.T) {
	_, err := BuildClient(
		"http://api.example.com",
		"tok_abc123",
		"warn",
	)
	if err == nil {
		t.Fatal("expected error for non-localhost HTTP")
	}
}

func TestConfigurationError_Message(t *testing.T) {
	err := &ConfigurationError{Message: "test error"}
	if err.Error() != "configuration error: test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestBuildClient_InvalidBaseURL(t *testing.T) {
	_, err := BuildClient(
		"not-a-url",
		"tok_abc123",
		"warn",
	)
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
