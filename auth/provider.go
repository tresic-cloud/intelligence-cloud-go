package auth

import (
	"context"
	"time"
)

// Token holds an access token and its expiry time. A zero-value ExpiresAt
// indicates that the token never expires — callers and caching layers treat
// this as an indefinitely valid credential.
type Token struct {
	// AccessToken is the bearer token presented in the Authorization header.
	// It is never logged or included in telemetry spans.
	AccessToken string

	// ExpiresAt is the time at which the token becomes invalid. A zero value
	// (time.Time{}) signals that the token does not expire.
	ExpiresAt time.Time
}

// CredentialProvider is the interface that credential adapters implement.
// The SDK client calls Token before each outbound request to obtain a
// bearer token. Implementations must be safe for concurrent use.
type CredentialProvider interface {
	// Token returns a valid access token or an error. The context carries
	// deadlines and cancellation signals that long-running providers
	// (e.g. those performing network calls) should honour.
	Token(ctx context.Context) (Token, error)
}
