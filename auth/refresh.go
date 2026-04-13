package auth

import (
	"context"
	"sync"
	"time"
)

// safetyMargin is the duration subtracted from Token.ExpiresAt when
// determining whether a cached token is still valid. This ensures the
// token is refreshed before it actually expires, avoiding edge-case
// failures caused by clock skew or network latency.
const safetyMargin = 30 * time.Second

// refreshFuncProvider wraps a caller-supplied refresh function with an
// in-memory cache that re-invokes the function only when the cached
// token has expired (or has never been fetched). Concurrent callers
// observe single-flight semantics: at most one invocation of fn runs
// at a time, and all waiters receive the result.
type refreshFuncProvider struct {
	fn func(ctx context.Context) (Token, error)

	mu     sync.Mutex
	cached Token
	valid  bool // true when cached holds a successfully fetched token
}

// Token returns a cached token if it is still valid, or invokes the
// wrapped refresh function to obtain a new one. If the refresh function
// returns an error, the cache is left untouched and the error is
// propagated to the caller. Concurrent calls that arrive while a
// refresh is in progress block on the mutex, ensuring exactly one
// invocation of the refresh function per cache miss.
func (p *refreshFuncProvider) Token(ctx context.Context) (Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.valid && !p.isExpired() {
		return p.cached, nil
	}

	tok, err := p.fn(ctx)
	if err != nil {
		return Token{}, err
	}

	p.cached = tok
	p.valid = true
	return tok, nil
}

// isExpired reports whether the cached token has passed its effective
// expiry (ExpiresAt minus the safety margin). A zero-value ExpiresAt
// means the token never expires.
//
// The caller must hold p.mu.
func (p *refreshFuncProvider) isExpired() bool {
	if p.cached.ExpiresAt.IsZero() {
		return false // zero-value = never expires
	}
	return time.Now().After(p.cached.ExpiresAt.Add(-safetyMargin))
}

// RefreshFunc returns a CredentialProvider that wraps the given refresh
// function with an in-memory token cache. The first call to Token
// invokes fn; subsequent calls return the cached token as long as it
// has not expired. The cache considers a token expired when the current
// time exceeds ExpiresAt minus a 30-second safety margin. A zero-value
// ExpiresAt is treated as "never expires," causing the token to be
// cached indefinitely.
//
// If fn returns an error, the error is surfaced to the caller and the
// cache is not updated, so the next call will retry fn.
//
// The returned provider is safe for concurrent use. When multiple
// goroutines call Token simultaneously during a cache miss, exactly one
// invocation of fn occurs (single-flight semantics via sync.Mutex).
func RefreshFunc(fn func(ctx context.Context) (Token, error)) CredentialProvider {
	return &refreshFuncProvider{fn: fn}
}
