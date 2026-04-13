package intelligencecloud

import (
	"fmt"
	"strings"
	"time"
)

// CallOption configures the behaviour of a single API call. Implementations
// apply themselves to an unexported callConfig that the transport layer reads.
type CallOption interface {
	applyCall(*callConfig)
}

// ListOption configures the behaviour of a paginated list call.
// Implementations apply themselves to an unexported listConfig that the
// iterator reads.
type ListOption interface {
	applyList(*listConfig)
}

// callConfig holds the resolved per-call configuration assembled from zero or
// more CallOption values. It is shared with the transport layer via
// package-internal access.
type callConfig struct {
	idempotencyKey string
	requestTimeout time.Duration
	extraHeaders   map[string]string
}

// listConfig holds the resolved per-list configuration assembled from zero or
// more ListOption values. It is shared with the iterator implementation via
// package-internal access.
type listConfig struct {
	pageSize int
	maxItems int
	filters  map[string]string
}

// ---------------------------------------------------------------------------
// CallOption helpers
// ---------------------------------------------------------------------------

type idempotencyKeyOption struct{ key string }

func (o idempotencyKeyOption) applyCall(c *callConfig) { c.idempotencyKey = o.key }

// WithIdempotencyKey returns a CallOption that sets the X-Idempotency-Key
// header on the outgoing request.
func WithIdempotencyKey(key string) CallOption {
	return idempotencyKeyOption{key: key}
}

type requestTimeoutOption struct{ d time.Duration }

func (o requestTimeoutOption) applyCall(c *callConfig) { c.requestTimeout = o.d }

// WithRequestTimeout returns a CallOption that applies a per-call timeout via
// context.WithTimeout, scoped to the individual request (not including
// retries).
func WithRequestTimeout(d time.Duration) CallOption {
	return requestTimeoutOption{d: d}
}

type extraHeaderOption struct {
	name  string
	value string
}

func (o extraHeaderOption) applyCall(c *callConfig) {
	if c.extraHeaders == nil {
		c.extraHeaders = make(map[string]string)
	}
	c.extraHeaders[o.name] = o.value
}

// WithExtraHeader returns a CallOption that adds an arbitrary header to the
// outgoing request. This is intended for future-proofing; common headers such
// as Authorization are managed by the SDK.
//
// WithExtraHeader panics if name is "Authorization" (case-insensitive),
// because the SDK manages that header via the CredentialProvider.
func WithExtraHeader(name, value string) CallOption {
	if strings.EqualFold(name, "authorization") {
		panic("intelligencecloud: WithExtraHeader: setting the Authorization header is forbidden; use a CredentialProvider instead")
	}
	return extraHeaderOption{name: name, value: value}
}

// ---------------------------------------------------------------------------
// ListOption helpers
// ---------------------------------------------------------------------------

type pageSizeOption struct{ n int }

func (o pageSizeOption) applyList(c *listConfig) { c.pageSize = o.n }

// WithPageSize returns a ListOption that hints the backend's page size
// (page_size query parameter). n must be > 0.
//
// WithPageSize panics if n <= 0.
func WithPageSize(n int) ListOption {
	if n <= 0 {
		panic(fmt.Sprintf("intelligencecloud: WithPageSize: n must be > 0, got %d", n))
	}
	return pageSizeOption{n: n}
}

type maxItemsOption struct{ n int }

func (o maxItemsOption) applyList(c *listConfig) { c.maxItems = o.n }

// WithMaxItems returns a ListOption that sets a hard cap on the number of
// items the iterator will return. n must be >= 0; 0 means unlimited.
//
// WithMaxItems panics if n < 0.
func WithMaxItems(n int) ListOption {
	if n < 0 {
		panic(fmt.Sprintf("intelligencecloud: WithMaxItems: n must be >= 0, got %d", n))
	}
	return maxItemsOption{n: n}
}

type filterOption struct {
	key   string
	value string
}

func (o filterOption) applyList(c *listConfig) {
	if c.filters == nil {
		c.filters = make(map[string]string)
	}
	c.filters[o.key] = o.value
}

// WithFilter returns a ListOption that adds a backend-defined filter parameter
// to the list request.
func WithFilter(key, value string) ListOption {
	return filterOption{key: key, value: value}
}
