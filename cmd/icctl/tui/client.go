// Package tui provides helpers for constructing SDK clients and other
// terminal-UI concerns from the CLI's resolved State.
package tui

import (
	"fmt"
	"log/slog"
	"os"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

// State mirrors cmd.State to avoid a circular import. At integration time,
// cmd.State is passed directly; this alias provides the field contract.
type State struct {
	EffectiveBaseURL string
	EffectiveToken   string
	LogLevel         string
	RequestID        string
	Timeout          string // duration string, kept for documentation; actual timeout is context-based
}

// BuildClient constructs an *intelligencecloud.Client from the resolved CLI
// state. It applies:
//   - EffectiveBaseURL as the client's base URL
//   - EffectiveToken via auth.StaticToken
//   - LogLevel as a slog.Logger at the corresponding level
//   - RequestID (if non-empty) is stored for callers to pass as a CallOption
//
// An empty EffectiveToken returns a configuration error (exit code 3 in the
// CLI's exit-code mapper). Timeout is applied as a context deadline by the
// calling command, not as an HTTP client timeout.
func BuildClient(baseURL, token, logLevel string) (*intelligencecloud.Client, error) {
	if token == "" {
		return nil, &ConfigurationError{
			Message: "no token available; set --token, ICCTL_TOKEN, or configure a profile with a stored token",
		}
	}

	if baseURL == "" {
		return nil, &ConfigurationError{
			Message: "no base URL available; set --base-url, ICCTL_BASE_URL, or configure a profile with an environment",
		}
	}

	// Build the slog logger at the requested level.
	logger := buildLogger(logLevel)

	client, err := intelligencecloud.NewClient(
		baseURL,
		auth.StaticToken(token),
		intelligencecloud.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct SDK client: %w", err)
	}

	return client, nil
}

// buildLogger creates a slog.Logger writing to stderr at the given level.
func buildLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn", "":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelWarn
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel,
	})
	return slog.New(handler)
}

// ConfigurationError represents a CLI configuration problem that maps to
// exit code 3 per cli-schema.md.
type ConfigurationError struct {
	Message string
}

// Error implements the error interface.
func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error: %s", e.Message)
}
