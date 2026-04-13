package transport

import (
	"math"
	"net/http"
	"testing"
	"time"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
)

// --- C-3: retry decision function tests ---

func TestShouldRetry_RetryableStatuses(t *testing.T) {
	t.Parallel()
	policy := ic.DefaultRetryPolicy()

	retryable := []int{429, 500, 502, 503, 504}
	for _, code := range retryable {
		resp := &http.Response{StatusCode: code}
		if !shouldRetry(policy, resp, nil) {
			t.Errorf("status %d should be retryable", code)
		}
	}
}

func TestShouldRetry_NonRetryableStatuses(t *testing.T) {
	t.Parallel()
	policy := ic.DefaultRetryPolicy()

	// 4xx codes that are not 429 should never be retried.
	nonRetryable := []int{400, 401, 403, 404, 409, 422}
	for _, code := range nonRetryable {
		resp := &http.Response{StatusCode: code}
		if shouldRetry(policy, resp, nil) {
			t.Errorf("status %d should NOT be retryable", code)
		}
	}
}

func TestShouldRetry_TransportError(t *testing.T) {
	t.Parallel()
	policy := ic.DefaultRetryPolicy()

	// Transport-level errors (nil response) should be retryable.
	if !shouldRetry(policy, nil, &http.MaxBytesError{}) {
		t.Error("transport error should be retryable")
	}
}

func TestShouldRetry_DisabledPolicy(t *testing.T) {
	t.Parallel()
	policy := ic.NoRetry()

	resp := &http.Response{StatusCode: 500}
	if shouldRetry(policy, resp, nil) {
		t.Error("disabled policy should not allow retry")
	}
}

func TestShouldRetry_SuccessNotRetried(t *testing.T) {
	t.Parallel()
	policy := ic.DefaultRetryPolicy()

	for _, code := range []int{200, 201, 204, 301, 302} {
		resp := &http.Response{StatusCode: code}
		if shouldRetry(policy, resp, nil) {
			t.Errorf("status %d should not be retried", code)
		}
	}
}

func TestParseRetryAfter_IntegerSeconds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"1", 1 * time.Second},
		{"5", 5 * time.Second},
		{"60", 60 * time.Second},
		{"0", 0},
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.value)
		if got != tt.expected {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.value, got, tt.expected)
		}
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()
	// Set a date 10 seconds from now.
	future := time.Now().Add(10 * time.Second)
	dateStr := future.UTC().Format(http.TimeFormat)

	got := parseRetryAfter(dateStr)
	// Allow some tolerance for clock skew.
	if got < 8*time.Second || got > 12*time.Second {
		t.Errorf("parseRetryAfter(%q) = %v, want ~10s", dateStr, got)
	}
}

func TestParseRetryAfter_HTTPDateInPast(t *testing.T) {
	t.Parallel()
	past := time.Now().Add(-10 * time.Second)
	dateStr := past.UTC().Format(http.TimeFormat)

	got := parseRetryAfter(dateStr)
	if got != 0 {
		t.Errorf("parseRetryAfter(past date) = %v, want 0", got)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	t.Parallel()
	tests := []string{"", "abc", "-1"}
	for _, v := range tests {
		got := parseRetryAfter(v)
		if got != 0 {
			t.Errorf("parseRetryAfter(%q) = %v, want 0", v, got)
		}
	}
}

func TestComputeBackoff_ExponentialClamped(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 10,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Jitter:      ic.NoJitter,
	}

	// Attempt 0: 100ms, Attempt 1: 200ms, Attempt 2: 400ms, ...
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
		3200 * time.Millisecond,
		5 * time.Second, // clamped
		5 * time.Second, // clamped
	}
	for i, want := range expected {
		got := computeBackoff(policy, i)
		if got != want {
			t.Errorf("computeBackoff(attempt=%d) = %v, want %v", i, got, want)
		}
	}
}

func TestComputeBackoff_FullJitterBounds(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Jitter:      ic.FullJitter,
	}

	// Run many iterations and verify all values fall in [0, delay).
	for attempt := 0; attempt < 4; attempt++ {
		baseDelay := time.Duration(float64(policy.BaseDelay) * math.Pow(2, float64(attempt)))
		if baseDelay > policy.MaxDelay {
			baseDelay = policy.MaxDelay
		}
		for i := 0; i < 100; i++ {
			got := computeBackoff(policy, attempt)
			if got < 0 || got >= baseDelay {
				t.Errorf("FullJitter: computeBackoff(attempt=%d) = %v, want [0, %v)", attempt, got, baseDelay)
			}
		}
	}
}

func TestComputeBackoff_EqualJitterBounds(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Jitter:      ic.EqualJitter,
	}

	for attempt := 0; attempt < 4; attempt++ {
		baseDelay := time.Duration(float64(policy.BaseDelay) * math.Pow(2, float64(attempt)))
		if baseDelay > policy.MaxDelay {
			baseDelay = policy.MaxDelay
		}
		half := baseDelay / 2
		for i := 0; i < 100; i++ {
			got := computeBackoff(policy, attempt)
			if got < half || got >= baseDelay {
				t.Errorf("EqualJitter: computeBackoff(attempt=%d) = %v, want [%v, %v)", attempt, got, half, baseDelay)
			}
		}
	}
}

func TestComputeBackoff_MaxDelayClamped(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 10,
		BaseDelay:   1 * time.Second,
		MaxDelay:    2 * time.Second,
		Jitter:      ic.NoJitter,
	}

	// At attempt 5, raw exponential would be 32s but should be clamped to 2s.
	got := computeBackoff(policy, 5)
	if got != 2*time.Second {
		t.Errorf("computeBackoff(attempt=5) = %v, want %v", got, 2*time.Second)
	}
}

func TestRetryAfterDelay_Honoured(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:          true,
		MaxAttempts:      4,
		BaseDelay:        100 * time.Millisecond,
		MaxDelay:         5 * time.Second,
		Jitter:           ic.NoJitter,
		HonourRetryAfter: true,
		RetryableStatus:  []int{429},
	}

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"3"}},
	}

	d := retryAfterDelay(policy, resp)
	if d != 3*time.Second {
		t.Errorf("retryAfterDelay = %v, want 3s", d)
	}
}

func TestRetryAfterDelay_ClampedByMaxDelay(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:          true,
		MaxAttempts:      4,
		BaseDelay:        100 * time.Millisecond,
		MaxDelay:         2 * time.Second,
		Jitter:           ic.NoJitter,
		HonourRetryAfter: true,
		RetryableStatus:  []int{429},
	}

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"10"}},
	}

	d := retryAfterDelay(policy, resp)
	if d != 2*time.Second {
		t.Errorf("retryAfterDelay = %v, want 2s (clamped)", d)
	}
}

func TestRetryAfterDelay_NotHonoured(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:          true,
		MaxAttempts:      4,
		BaseDelay:        100 * time.Millisecond,
		MaxDelay:         5 * time.Second,
		Jitter:           ic.NoJitter,
		HonourRetryAfter: false,
		RetryableStatus:  []int{429},
	}

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"3"}},
	}

	d := retryAfterDelay(policy, resp)
	if d != 0 {
		t.Errorf("retryAfterDelay with HonourRetryAfter=false = %v, want 0", d)
	}
}

func TestRetryAfterDelay_NoHeader(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:          true,
		HonourRetryAfter: true,
		MaxDelay:         5 * time.Second,
	}

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}

	d := retryAfterDelay(policy, resp)
	if d != 0 {
		t.Errorf("retryAfterDelay with no header = %v, want 0", d)
	}
}

func TestComputeBackoff_EqualJitter_TinyDelay(t *testing.T) {
	t.Parallel()
	// When delay is 1ns, half is 0, EqualJitter should return 0.
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 4,
		BaseDelay:   1 * time.Nanosecond,
		MaxDelay:    1 * time.Nanosecond,
		Jitter:      ic.EqualJitter,
	}
	got := computeBackoff(policy, 0)
	if got != 0 {
		t.Errorf("computeBackoff(EqualJitter, 1ns) = %v, want 0", got)
	}
}

func TestComputeBackoff_FullJitter_TinyDelay(t *testing.T) {
	t.Parallel()
	policy := ic.RetryPolicy{
		Enabled:     true,
		MaxAttempts: 4,
		BaseDelay:   1 * time.Nanosecond,
		MaxDelay:    1 * time.Nanosecond,
		Jitter:      ic.FullJitter,
	}
	got := computeBackoff(policy, 0)
	if got != 0 {
		t.Errorf("computeBackoff(FullJitter, 1ns) = %v, want 0", got)
	}
}
