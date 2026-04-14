package intelligencecloud

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/version"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Client is the top-level handle for the Intelligence Cloud API. It is
// immutable after construction and safe for concurrent use by multiple
// goroutines. Per-call configuration (timeouts, idempotency keys, extra
// headers) is supplied via CallOption or ListOption variadic parameters on
// each resource-service method, not by mutating the Client.
//
// Construct a Client with NewClient; the zero value is not usable.
type Client struct {
	// unexported configuration fields — set once during construction.
	baseURL            *url.URL
	credentialProvider auth.CredentialProvider
	httpClient         *http.Client
	retryPolicy        RetryPolicy
	tracerProvider     trace.TracerProvider
	propagator         propagation.TextMapPropagator
	logger             *slog.Logger
	userAgent          string

	// Resource services — populated during construction. Each service holds
	// a non-owning reference back to the Client for HTTP execution. Method
	// implementations are added by resource-wrapper tasks in Wave 2.

	// Me provides access to the authenticated user's profile and preferences.
	// Populated by resource-wrapper tasks in Wave 2.
	Me *MeService

	// Resellers provides CRUD and lifecycle operations for reseller resources.
	// Populated by resource-wrapper tasks in Wave 2.
	Resellers *ResellerService

	// Products provides CRUD and lifecycle operations for product resources.
	// Populated by resource-wrapper tasks in Wave 2.
	Products *ProductService

	// Connectors provides CRUD and lifecycle operations for connector resources.
	// Populated by resource-wrapper tasks in Wave 2.
	Connectors *ConnectorService

	// AuditLogs provides read-only access to audit log entries.
	// Populated by resource-wrapper tasks in Wave 2.
	AuditLogs *AuditLogService

	// Users provides CRUD and lifecycle operations for user resources.
	// Populated by resource-wrapper tasks in Wave 2.
	Users *UserService

	// Auth provides authentication-related operations (e.g. token exchange).
	// Populated by resource-wrapper tasks in Wave 2.
	Auth *AuthService
}

// ---------------------------------------------------------------------------
// Resource service stubs — Wave 2 extends these with API methods.
// ---------------------------------------------------------------------------

// MeService provides access to the authenticated user's profile and
// notification preferences. Methods are added by resource-wrapper tasks in
// Wave 2.
type MeService struct{ client *Client }

// ResellerService provides CRUD and lifecycle operations for reseller
// resources. Methods are added by resource-wrapper tasks in Wave 2.
type ResellerService struct{ client *Client }

// ProductService provides CRUD and lifecycle operations for product
// resources. Methods are added by resource-wrapper tasks in Wave 2.
type ProductService struct{ client *Client }

// ConnectorService provides CRUD and lifecycle operations for connector
// resources. Methods are added by resource-wrapper tasks in Wave 2.
type ConnectorService struct{ client *Client }

// AuditLogService provides read-only access to audit log entries. Methods
// are added by resource-wrapper tasks in Wave 2.
type AuditLogService struct{ client *Client }

// UserService provides CRUD and lifecycle operations for user resources.
// Methods are added by resource-wrapper tasks in Wave 2.
type UserService struct{ client *Client }

// AuthService provides authentication-related operations such as token
// exchange. Methods are added by resource-wrapper tasks in Wave 2.
type AuthService struct{ client *Client }

// ---------------------------------------------------------------------------
// ClientOption
// ---------------------------------------------------------------------------

// ClientOption configures the Client during construction. Implementations
// apply themselves to the Client's unexported fields. The interface is open
// so that future releases may add new options without breaking existing
// callers.
type ClientOption interface {
	applyClient(*Client)
}

// ---------------------------------------------------------------------------
// ClientOption helpers
// ---------------------------------------------------------------------------

type httpClientOption struct{ c *http.Client }

func (o httpClientOption) applyClient(cl *Client) { cl.httpClient = o.c }

// WithHTTPClient returns a ClientOption that sets the underlying HTTP client
// used for all API requests. When not supplied, NewClient creates a default
// client with a 30-second timeout. Transport wrapping (retry, tracing, auth
// injection) is applied additively by the transport layer.
func WithHTTPClient(c *http.Client) ClientOption {
	return httpClientOption{c: c}
}

type retryPolicyOption struct{ p RetryPolicy }

func (o retryPolicyOption) applyClient(cl *Client) { cl.retryPolicy = o.p }

// WithRetryPolicy returns a ClientOption that sets the retry policy. The
// policy is validated during NewClient construction; invalid policies cause
// NewClient to return a *ConfigurationError.
func WithRetryPolicy(p RetryPolicy) ClientOption {
	return retryPolicyOption{p: p}
}

type disableRetryOption struct{}

func (o disableRetryOption) applyClient(cl *Client) { cl.retryPolicy = NoRetry() }

// WithDisableRetry returns a ClientOption that disables automatic retries.
// This is equivalent to WithRetryPolicy(NoRetry()) and is provided as a
// convenience.
func WithDisableRetry() ClientOption {
	return disableRetryOption{}
}

type tracerProviderOption struct{ tp trace.TracerProvider }

func (o tracerProviderOption) applyClient(cl *Client) { cl.tracerProvider = o.tp }

// WithTracerProvider returns a ClientOption that sets the OpenTelemetry
// TracerProvider for distributed tracing. When not supplied, the Client
// uses a no-op tracer provider that allocates nothing.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return tracerProviderOption{tp: tp}
}

type propagatorOption struct{ p propagation.TextMapPropagator }

func (o propagatorOption) applyClient(cl *Client) { cl.propagator = o.p }

// WithPropagator returns a ClientOption that sets the trace-context
// propagator. When not supplied, the Client uses a composite propagator
// with W3C TraceContext and Baggage.
func WithPropagator(p propagation.TextMapPropagator) ClientOption {
	return propagatorOption{p: p}
}

type loggerOption struct{ l *slog.Logger }

func (o loggerOption) applyClient(cl *Client) { cl.logger = o.l }

// WithLogger returns a ClientOption that sets the structured logger. When
// not supplied, the Client uses a logger backed by a discarding handler so
// that no log output is produced.
func WithLogger(l *slog.Logger) ClientOption {
	return loggerOption{l: l}
}

type userAgentOption struct{ ua string }

func (o userAgentOption) applyClient(cl *Client) { cl.userAgent = o.ua }

// WithUserAgent returns a ClientOption that overrides the default
// User-Agent header value. The default format is
// "intelligence-cloud-go/<version> (<os>/<arch>)".
func WithUserAgent(ua string) ClientOption {
	return userAgentOption{ua: ua}
}

// ---------------------------------------------------------------------------
// NewClient
// ---------------------------------------------------------------------------

// NewClient constructs a Client for the Intelligence Cloud API. baseURL is
// the root URL of the API (must use https:// unless the host is localhost or
// 127.0.0.1). provider supplies bearer tokens for authentication and must
// not be nil. Optional ClientOption values configure HTTP client, retry
// policy, tracing, logging, and user-agent overrides.
//
// NewClient returns a *ConfigurationError (which does NOT implement
// APIError) when inputs are invalid. The returned Client is immutable and
// safe for concurrent use.
func NewClient(baseURL string, provider auth.CredentialProvider, opts ...ClientOption) (*Client, error) {
	// Validate provider first — a nil provider is always an error.
	if provider == nil {
		return nil, &ConfigurationError{Message: "credential provider must not be nil"}
	}

	// Parse and validate the base URL.
	parsedURL, err := parseAndValidateBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	// Construct client with defaults.
	c := &Client{
		baseURL:            parsedURL,
		credentialProvider: provider,
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		retryPolicy:        DefaultRetryPolicy(),
		tracerProvider:     noop.NewTracerProvider(),
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
		logger:    slog.New(discardHandler{}),
		userAgent: defaultUserAgent(),
	}

	// Apply options.
	for _, opt := range opts {
		opt.applyClient(c)
	}

	// Validate retry policy after options are applied.
	if err := c.retryPolicy.Validate(); err != nil {
		return nil, err
	}

	// Wire resource service stubs.
	c.Me = &MeService{client: c}
	c.Resellers = &ResellerService{client: c}
	c.Products = &ProductService{client: c}
	c.Connectors = &ConnectorService{client: c}
	c.AuditLogs = &AuditLogService{client: c}
	c.Users = &UserService{client: c}
	c.Auth = &AuthService{client: c}

	return c, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// parseAndValidateBaseURL parses rawURL and enforces the SDK's URL scheme
// rules: https is required unless the host is localhost or 127.0.0.1.
func parseAndValidateBaseURL(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, &ConfigurationError{Message: "baseURL must not be empty"}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("baseURL is not a valid URL: %v", err),
		}
	}

	// url.Parse is very permissive — a string like "not a url" parses
	// successfully with an empty scheme. Require a scheme explicitly.
	if u.Scheme == "" {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("baseURL %q must include a scheme (https://)", rawURL),
		}
	}

	// Only https and http are accepted schemes.
	switch u.Scheme {
	case "https":
		// Always accepted.
	case "http":
		// HTTP is only permitted for localhost or loopback addresses.
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" {
			return nil, &ConfigurationError{
				Message: fmt.Sprintf("baseURL %q uses http:// which is only permitted for localhost or 127.0.0.1", rawURL),
			}
		}
	default:
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("baseURL %q must use https:// (or http:// for localhost)", rawURL),
		}
	}

	return u, nil
}

// defaultUserAgent returns the default User-Agent header value using the
// build-time version from internal/version and the current OS/architecture.
func defaultUserAgent() string {
	return fmt.Sprintf("intelligence-cloud-go/%s (%s/%s)",
		version.Version, runtime.GOOS, runtime.GOARCH)
}

// discardHandler is an slog.Handler that discards all log records. It is
// used as the default logger handler when no logger is provided via
// WithLogger, ensuring the Client never produces log output unless the
// caller opts in.
type discardHandler struct{}

func (discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (discardHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return discardHandler{} }
func (discardHandler) WithGroup(_ string) slog.Handler               { return discardHandler{} }
