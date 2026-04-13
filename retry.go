package intelligencecloud

import "time"

// JitterStrategy controls how jitter is applied to retry backoff delays.
type JitterStrategy int

const (
	// NoJitter applies no jitter; the backoff delay is used exactly.
	NoJitter JitterStrategy = iota
	// EqualJitter applies jitter of up to half the computed delay.
	EqualJitter
	// FullJitter applies jitter across the full range [0, delay).
	FullJitter
)

// RetryPolicy controls how the SDK retries transient failures. The zero value
// is not usable; use DefaultRetryPolicy or NoRetry to obtain a valid policy.
// The retry execution loop lives in internal/transport; this file defines
// only the policy struct, factories, and validation.
type RetryPolicy struct {
	// Enabled controls whether retries are attempted at all.
	Enabled bool
	// MaxAttempts is the total number of attempts (1 initial + N-1 retries).
	// Must be >= 1.
	MaxAttempts int
	// BaseDelay is the initial backoff delay before the first retry. Must be
	// > 0.
	BaseDelay time.Duration
	// MaxDelay caps the computed backoff delay for any single retry. Must be
	// >= BaseDelay.
	MaxDelay time.Duration
	// Jitter selects the jitter strategy applied to the backoff delay.
	Jitter JitterStrategy
	// RetryableStatus lists the HTTP status codes that are eligible for retry.
	RetryableStatus []int
	// HonourRetryAfter controls whether the SDK respects Retry-After headers
	// from 429 responses, overriding the computed backoff for that attempt.
	HonourRetryAfter bool
}

// DefaultRetryPolicy returns the recommended retry policy for production use.
// It enables retries with 4 total attempts (1 initial + 3 retries), 100 ms
// base delay, 30 s maximum delay, full jitter, and honours Retry-After
// headers on 429 responses.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Enabled:          true,
		MaxAttempts:      4,
		BaseDelay:        100 * time.Millisecond,
		MaxDelay:         30 * time.Second,
		Jitter:           FullJitter,
		RetryableStatus:  []int{429, 500, 502, 503, 504},
		HonourRetryAfter: true,
	}
}

// NoRetry returns a policy that disables all retries.
func NoRetry() RetryPolicy {
	return RetryPolicy{
		Enabled:     false,
		MaxAttempts: 1,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Jitter:      NoJitter,
	}
}

// Validate checks that the policy fields are internally consistent. It returns
// a *ConfigurationError if any rule is violated. Validation rules apply
// regardless of the Enabled flag so that a disabled policy can be re-enabled
// safely.
func (p RetryPolicy) Validate() error {
	if p.MaxAttempts < 1 {
		return &ConfigurationError{Message: "RetryPolicy.MaxAttempts must be >= 1"}
	}
	if p.BaseDelay <= 0 {
		return &ConfigurationError{Message: "RetryPolicy.BaseDelay must be > 0"}
	}
	if p.MaxDelay < p.BaseDelay {
		return &ConfigurationError{Message: "RetryPolicy.MaxDelay must be >= BaseDelay"}
	}
	return nil
}
