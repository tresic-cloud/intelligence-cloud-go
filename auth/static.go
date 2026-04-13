package auth

import (
	"context"
	"time"
)

// staticTokenProvider is a CredentialProvider that always returns the same
// token with a zero expiry (never expires). It is safe for concurrent use
// because it holds no mutable state.
type staticTokenProvider struct {
	token Token
}

// Token returns the static token and a nil error. The context is accepted
// for interface compliance but is not consulted.
func (p *staticTokenProvider) Token(_ context.Context) (Token, error) {
	return p.token, nil
}

// StaticToken returns a CredentialProvider that always yields the given
// access token with a zero-value ExpiresAt (never expires). The empty
// string is accepted without panic. The returned provider is safe for
// concurrent use by multiple goroutines.
func StaticToken(s string) CredentialProvider {
	return &staticTokenProvider{
		token: Token{
			AccessToken: s,
			ExpiresAt:   time.Time{},
		},
	}
}
