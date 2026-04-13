// Package intelligencecloud provides a typed Go client for the Intelligence
// Cloud API. It exposes resource services, typed errors, a configurable retry
// policy, and per-call/per-list options.
//
// # Quick start
//
//	client, err := intelligencecloud.NewClient(
//	    "https://api.intelligencecloud.example.com",
//	    auth.StaticToken("my-bearer-token"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Client construction
//
// NewClient is the entry point. It requires a base URL (https:// unless the
// host is localhost or 127.0.0.1) and an auth.CredentialProvider. Optional
// ClientOption values configure the HTTP client, retry policy, tracing,
// logging, and User-Agent header.
//
// The returned Client is immutable after construction and safe for concurrent
// use by multiple goroutines.
//
// # Resource services
//
// Each resource type (Resellers, Companies, Locations, etc.) is accessed as
// a field on the Client. Resource services provide CRUD and lifecycle methods
// that return typed responses or errors.
//
// # Error handling
//
// API errors implement the APIError interface and wrap sentinel errors for
// use with errors.Is. Callers can use errors.As to access concrete error
// types with additional fields.
//
// Configuration errors (returned only from NewClient) are *ConfigurationError
// and do NOT implement APIError.
//
// # Pagination
//
// List methods return an Iterator[T] that lazily fetches pages. Use
// WithPageSize and WithMaxItems to control pagination behaviour.
//
// # Retry policy
//
// By default the client retries transient failures (429, 5xx, transport
// errors) with exponential backoff and full jitter. Use WithRetryPolicy or
// WithDisableRetry to customise this behaviour.
//
// # Observability
//
// Tracing is supported via OpenTelemetry. Use WithTracerProvider and
// WithPropagator to enable distributed tracing. Use WithLogger to enable
// structured logging with automatic PII redaction.
package intelligencecloud
