package transport

import (
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
)

// shouldRetry determines whether the given response/error combination is
// eligible for retry under the provided policy. It does NOT handle 401
// responses (those use a separate re-fetch path) or budget exhaustion.
func shouldRetry(policy ic.RetryPolicy, resp *http.Response, err error) bool {
	if !policy.Enabled {
		return false
	}

	// Transport-level errors (DNS, connection, TLS) are always retryable.
	if err != nil {
		return true
	}

	if resp == nil {
		return false
	}

	for _, code := range policy.RetryableStatus {
		if resp.StatusCode == code {
			return true
		}
	}

	return false
}

// parseRetryAfter parses the value of a Retry-After header. It supports
// integer-seconds ("120") and HTTP-date ("Mon, 02 Jan 2006 15:04:05 GMT")
// formats. Returns 0 if the value cannot be parsed or refers to a time
// in the past.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try integer-seconds first (most common).
	if secs, err := strconv.Atoi(value); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}

	// Try HTTP-date format.
	t, err := http.ParseTime(value)
	if err != nil {
		return 0
	}

	d := time.Until(t)
	if d <= 0 {
		return 0
	}
	return d
}

// retryAfterDelay returns the Retry-After delay from the response, clamped
// by policy.MaxDelay. Returns 0 if HonourRetryAfter is false, the header
// is missing, or the header cannot be parsed.
func retryAfterDelay(policy ic.RetryPolicy, resp *http.Response) time.Duration {
	if !policy.HonourRetryAfter {
		return 0
	}
	if resp == nil {
		return 0
	}
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 0
	}
	d := parseRetryAfter(val)
	if d > policy.MaxDelay {
		d = policy.MaxDelay
	}
	return d
}

// computeBackoff computes the backoff delay for the given attempt index
// (0-based, where attempt 0 is the first retry). Uses exponential backoff
// with the configured jitter strategy and MaxDelay clamp.
func computeBackoff(policy ic.RetryPolicy, attempt int) time.Duration {
	// Exponential: baseDelay * 2^attempt
	delay := time.Duration(float64(policy.BaseDelay) * math.Pow(2, float64(attempt)))
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	switch policy.Jitter {
	case ic.FullJitter:
		// Random value in [0, delay)
		if delay <= 0 {
			return 0
		}
		return time.Duration(rand.Int64N(int64(delay)))
	case ic.EqualJitter:
		// Random value in [delay/2, delay)
		half := delay / 2
		if half <= 0 {
			return 0
		}
		jitter := time.Duration(rand.Int64N(int64(half)))
		return half + jitter
	default: // NoJitter
		return delay
	}
}

// retryCause returns a string describing why a retry is being attempted,
// suitable for span attributes and log records.
func retryCause(resp *http.Response, err error) string {
	if err != nil {
		return "transport_error"
	}
	if resp != nil {
		return "status_" + strconv.Itoa(resp.StatusCode)
	}
	return "unknown"
}
