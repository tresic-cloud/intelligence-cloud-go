// Package cmd implements the cobra command tree for the icctl CLI binary.
// It provides the root command, global flag resolution, version subcommand,
// profile subcommand tree, and context-based state propagation.
package cmd

import (
	"context"
	"time"
)

// State holds the resolved, non-persistent configuration for a single CLI
// invocation. It is built during PersistentPreRunE after merging flag, env,
// profile, and built-in defaults per the 4-layer precedence rule.
type State struct {
	// ProfileName is the name of the active profile.
	ProfileName string

	// EffectiveBaseURL is the resolved API base URL after precedence.
	EffectiveBaseURL string

	// EffectiveToken is the resolved bearer token after precedence.
	EffectiveToken string

	// EffectiveOutput is the resolved output format ("table" or "json").
	EffectiveOutput string

	// EffectiveEnvironment is the resolved environment name.
	EffectiveEnvironment string

	// AssumeYes indicates whether destructive verbs should skip confirmation.
	AssumeYes bool

	// LogLevel is the resolved log level string ("debug", "info", "warn", "error").
	LogLevel string

	// RequestID is the forced request ID, or empty for auto-generated.
	RequestID string

	// Timeout is the per-call timeout duration.
	Timeout time.Duration
}

// stateKey is the unexported context key for State.
type stateKey struct{}

// WithState returns a new context carrying the given State.
func WithState(ctx context.Context, s *State) context.Context {
	return context.WithValue(ctx, stateKey{}, s)
}

// StateFrom extracts the State from the context. It returns nil if no State
// is present.
func StateFrom(ctx context.Context) *State {
	s, _ := ctx.Value(stateKey{}).(*State)
	return s
}
