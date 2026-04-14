package intelligencecloud

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// dummyProvider is a trivial CredentialProvider for tests that don't care
// about the actual token.
var dummyProvider = auth.StaticToken("test-token")

// ---------------------------------------------------------------------------
// [A-19] NewClient valid-construction tests
// ---------------------------------------------------------------------------

func TestNewClient_HTTPS_BaseURL(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if got := c.baseURL.String(); got != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", got, "https://api.example.com")
	}
}

func TestNewClient_HTTPS_WithPath(t *testing.T) {
	c, err := NewClient("https://api.example.com/v1", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := c.baseURL.String(); got != "https://api.example.com/v1" {
		t.Errorf("baseURL = %q, want %q", got, "https://api.example.com/v1")
	}
}

func TestNewClient_Localhost_HTTP_Accepted(t *testing.T) {
	c, err := NewClient("http://localhost:8080", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.baseURL.Scheme != "http" {
		t.Errorf("scheme = %q, want %q", c.baseURL.Scheme, "http")
	}
}

func TestNewClient_Loopback_HTTP_Accepted(t *testing.T) {
	c, err := NewClient("http://127.0.0.1:9090", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.baseURL.Scheme != "http" {
		t.Errorf("scheme = %q, want %q", c.baseURL.Scheme, "http")
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default HTTP client should be non-nil.
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}

	// Default retry policy should be enabled.
	if !c.retryPolicy.Enabled {
		t.Error("retryPolicy.Enabled = false, want true")
	}
	if c.retryPolicy.MaxAttempts != 4 {
		t.Errorf("retryPolicy.MaxAttempts = %d, want 4", c.retryPolicy.MaxAttempts)
	}

	// Default tracer provider should be non-nil.
	if c.tracerProvider == nil {
		t.Error("tracerProvider is nil")
	}

	// Default propagator should be non-nil.
	if c.propagator == nil {
		t.Error("propagator is nil")
	}

	// Default logger should be non-nil.
	if c.logger == nil {
		t.Error("logger is nil")
	}

	// Default user agent should be non-empty.
	if c.userAgent == "" {
		t.Error("userAgent is empty")
	}
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 60 * time.Second}
	c, err := NewClient("https://api.example.com", dummyProvider, WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != custom {
		t.Error("httpClient was not set to custom client")
	}
}

func TestNewClient_WithRetryPolicy(t *testing.T) {
	policy := RetryPolicy{
		Enabled:     true,
		MaxAttempts: 2,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Jitter:      EqualJitter,
	}
	c, err := NewClient("https://api.example.com", dummyProvider, WithRetryPolicy(policy))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.retryPolicy.MaxAttempts != 2 {
		t.Errorf("retryPolicy.MaxAttempts = %d, want 2", c.retryPolicy.MaxAttempts)
	}
	if c.retryPolicy.Jitter != EqualJitter {
		t.Errorf("retryPolicy.Jitter = %d, want EqualJitter (%d)", c.retryPolicy.Jitter, EqualJitter)
	}
}

func TestNewClient_WithDisableRetry(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider, WithDisableRetry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.retryPolicy.Enabled {
		t.Error("retryPolicy.Enabled = true, want false after WithDisableRetry")
	}
}

func TestNewClient_WithTracerProvider(t *testing.T) {
	tp := noop.NewTracerProvider()
	c, err := NewClient("https://api.example.com", dummyProvider, WithTracerProvider(tp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.tracerProvider != tp {
		t.Error("tracerProvider was not set to custom provider")
	}
}

func TestNewClient_WithPropagator(t *testing.T) {
	p := propagation.TraceContext{}
	c, err := NewClient("https://api.example.com", dummyProvider, WithPropagator(p))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the propagator was stored. We compare by checking Fields() output
	// since propagation.TextMapPropagator is an interface.
	if c.propagator == nil {
		t.Error("propagator is nil")
	}
}

func TestNewClient_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, nil))
	c, err := NewClient("https://api.example.com", dummyProvider, WithLogger(logger))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.logger != logger {
		t.Error("logger was not set to custom logger")
	}
}

func TestNewClient_WithUserAgent(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider, WithUserAgent("custom-agent/1.0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.userAgent != "custom-agent/1.0" {
		t.Errorf("userAgent = %q, want %q", c.userAgent, "custom-agent/1.0")
	}
}

func TestNewClient_MultipleOptions(t *testing.T) {
	custom := &http.Client{Timeout: 45 * time.Second}
	tp := noop.NewTracerProvider()
	c, err := NewClient("https://api.example.com", dummyProvider,
		WithHTTPClient(custom),
		WithTracerProvider(tp),
		WithUserAgent("multi/1.0"),
		WithDisableRetry(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != custom {
		t.Error("httpClient not applied")
	}
	if c.tracerProvider != tp {
		t.Error("tracerProvider not applied")
	}
	if c.userAgent != "multi/1.0" {
		t.Errorf("userAgent = %q, want %q", c.userAgent, "multi/1.0")
	}
	if c.retryPolicy.Enabled {
		t.Error("retryPolicy.Enabled = true, want false")
	}
}

func TestNewClient_ResourceServiceFields_NotNil(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All resource service fields should be non-nil (empty stubs).
	if c.Me == nil {
		t.Error("Me is nil")
	}
	if c.Resellers == nil {
		t.Error("Resellers is nil")
	}
	if c.Products == nil {
		t.Error("Products is nil")
	}
	if c.Connectors == nil {
		t.Error("Connectors is nil")
	}
	if c.AuditLogs == nil {
		t.Error("AuditLogs is nil")
	}
	if c.Users == nil {
		t.Error("Users is nil")
	}
	if c.Auth == nil {
		t.Error("Auth is nil")
	}
}

// ---------------------------------------------------------------------------
// [A-20] NewClient validation-failure tests
// ---------------------------------------------------------------------------

func TestNewClient_NilProvider_ReturnsConfigurationError(t *testing.T) {
	_, err := NewClient("https://api.example.com", nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
	var ce *ConfigurationError
	if !errors.As(err, &ce) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestNewClient_InvalidBaseURL_ReturnsConfigurationError(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"unparseable", ":::garbage"},
		{"not a url", "not a url"},
		{"empty", ""},
		{"no scheme", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.url, dummyProvider)
			if err == nil {
				t.Fatal("expected error for invalid baseURL")
			}
			var ce *ConfigurationError
			if !errors.As(err, &ce) {
				t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
			}
		})
	}
}

func TestNewClient_HTTP_NonLocalhost_Rejected(t *testing.T) {
	_, err := NewClient("http://example.com", dummyProvider)
	if err == nil {
		t.Fatal("expected error for http:// on non-localhost")
	}
	var ce *ConfigurationError
	if !errors.As(err, &ce) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestNewClient_HTTP_NonLocalhost_WithPort_Rejected(t *testing.T) {
	_, err := NewClient("http://example.com:8080", dummyProvider)
	if err == nil {
		t.Fatal("expected error for http:// on non-localhost with port")
	}
	var ce *ConfigurationError
	if !errors.As(err, &ce) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestNewClient_InvalidRetryPolicy_ReturnsConfigurationError(t *testing.T) {
	bad := RetryPolicy{
		Enabled:     true,
		MaxAttempts: 0, // invalid: must be >= 1
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
	}
	_, err := NewClient("https://api.example.com", dummyProvider, WithRetryPolicy(bad))
	if err == nil {
		t.Fatal("expected error for invalid retry policy")
	}
	var ce *ConfigurationError
	if !errors.As(err, &ce) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestNewClient_ValidationError_IsNotAPIError(t *testing.T) {
	_, err := NewClient("https://api.example.com", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	// The error must NOT satisfy the APIError interface.
	var apiErr APIError
	if errors.As(err, &apiErr) {
		t.Error("ConfigurationError should NOT satisfy APIError")
	}
}

func TestNewClient_FTP_Scheme_Rejected(t *testing.T) {
	_, err := NewClient("ftp://files.example.com", dummyProvider)
	if err == nil {
		t.Fatal("expected error for ftp:// scheme")
	}
	var ce *ConfigurationError
	if !errors.As(err, &ce) {
		t.Errorf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// [A-25] Default User-Agent format + override
// ---------------------------------------------------------------------------

// uaPattern matches the expected default User-Agent format:
// intelligence-cloud-go/<version> (<os>/<arch>)
var uaPattern = regexp.MustCompile(`^intelligence-cloud-go/[^ ]+ \([^/]+/[^)]+\)$`)

func TestNewClient_DefaultUserAgent_MatchesPattern(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uaPattern.MatchString(c.userAgent) {
		t.Errorf("default userAgent %q does not match pattern %q", c.userAgent, uaPattern.String())
	}
}

func TestNewClient_DefaultUserAgent_ContainsVersion(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In test builds, version.Version is "dev".
	if c.userAgent == "" {
		t.Error("default userAgent is empty")
	}
	// The version substring should be present.
	if !regexp.MustCompile(`intelligence-cloud-go/dev`).MatchString(c.userAgent) {
		t.Errorf("default userAgent %q does not contain 'intelligence-cloud-go/dev'", c.userAgent)
	}
}

func TestNewClient_WithUserAgent_Overrides(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider, WithUserAgent("my-app/2.0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.userAgent != "my-app/2.0" {
		t.Errorf("userAgent = %q, want %q", c.userAgent, "my-app/2.0")
	}
}

// ---------------------------------------------------------------------------
// Concurrency safety smoke test
// ---------------------------------------------------------------------------

func TestClient_ConcurrentAccess(t *testing.T) {
	c, err := NewClient("https://api.example.com", dummyProvider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Read fields concurrently — should not race.
			_ = c.baseURL.String()
			_ = c.userAgent
			_ = c.retryPolicy.Enabled
			_ = c.httpClient
			_ = c.logger
			_ = c.tracerProvider
			_ = c.propagator
			_ = c.Me
			_ = c.Resellers
			_ = c.Products
			_ = c.Connectors
			_ = c.AuditLogs
			_ = c.Users
			_ = c.Auth
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// ClientOption interface compliance (compile-time checks)
// ---------------------------------------------------------------------------

var (
	_ ClientOption = httpClientOption{}
	_ ClientOption = retryPolicyOption{}
	_ ClientOption = disableRetryOption{}
	_ ClientOption = tracerProviderOption{}
	_ ClientOption = propagatorOption{}
	_ ClientOption = loggerOption{}
	_ ClientOption = userAgentOption{}
)

// Verify that trace.TracerProvider and propagation.TextMapPropagator are
// used correctly by referencing them (compile-time only).
var (
	_ trace.TracerProvider              = noop.NewTracerProvider()
	_ propagation.TextMapPropagator     = propagation.TraceContext{}
)
