package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	ic "github.com/tresic-cloud/intelligence-cloud-go"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// --- C-12: Integration tests with in-memory OTel exporter ---

// newIntegrationTransport creates a fully wired Transport with an in-memory
// OTel exporter for integration testing.
func newIntegrationTransport(
	t *testing.T,
	handler http.Handler,
	provider *countingProvider,
	policy ic.RetryPolicy,
) (*Transport, *httptest.Server, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { tp.Shutdown(context.Background()) })

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tr := &Transport{
		Base:       srv.Client().Transport,
		Provider:   provider,
		Policy:     policy,
		TracerProv: tp,
		Propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
		UserAgent: "integration-test/1.0",
	}

	return tr, srv, exp
}

func TestIntegration_HappyPath(t *testing.T) {
	t.Parallel()
	provider := &countingProvider{tokens: []string{"valid-tok"}}

	tr, srv, exp := newIntegrationTransport(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify auth header.
			if r.Header.Get("Authorization") != "Bearer valid-tok" {
				t.Errorf("missing bearer token")
			}
			// Verify UA.
			if r.Header.Get("User-Agent") != "integration-test/1.0" {
				t.Errorf("wrong UA: %q", r.Header.Get("User-Agent"))
			}
			// Verify propagation headers.
			if r.Header.Get("Traceparent") == "" {
				t.Error("missing traceparent header")
			}

			w.Header().Set("X-Request-Id", "int-req-1")
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"ok"}`))
		}),
		provider,
		noRetryPolicy(),
	)

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "GetMe",
		URLTemplate: "/api/v1/me",
		Resource:    "me",
	})
	req := newRequest(t, "GET", srv.URL+"/me", nil)
	req = req.WithContext(ctx)

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "intelligencecloud.GetMe" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "intelligencecloud.GetMe")
	}
	if spans[0].Status.Code != codes.Unset {
		t.Errorf("span status = %v, want Unset", spans[0].Status.Code)
	}
}

func TestIntegration_RetryThenSuccess(t *testing.T) {
	t.Parallel()
	callNum := int32(0)
	provider := &countingProvider{tokens: []string{"tok"}}

	tr, srv, exp := newIntegrationTransport(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := atomic.AddInt32(&callNum, 1)
			if n <= 2 {
				w.Header().Set("X-Request-Id", "int-retry")
				w.WriteHeader(503)
				return
			}
			w.Header().Set("X-Request-Id", "int-retry-ok")
			w.WriteHeader(200)
		}),
		provider,
		fastRetryPolicy(),
	)

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "ListResellers",
		URLTemplate: "/api/v1/resellers",
		Resource:    "resellers",
	})
	req := newRequest(t, "GET", srv.URL+"/resellers", nil)
	req = req.WithContext(ctx)

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Should have retry.attempt_failed events.
	failedEvents := 0
	for _, ev := range spans[0].Events {
		if ev.Name == "retry.attempt_failed" {
			failedEvents++
		}
	}
	if failedEvents != 2 {
		t.Errorf("retry.attempt_failed events = %d, want 2", failedEvents)
	}

	// 3 actual server calls.
	if n := atomic.LoadInt32(&callNum); n != 3 {
		t.Errorf("server calls = %d, want 3", n)
	}
}

func TestIntegration_RetriesExhausted(t *testing.T) {
	t.Parallel()
	provider := &countingProvider{tokens: []string{"tok"}}

	tr, srv, exp := newIntegrationTransport(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Request-Id", "int-exhaust")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]any{
				"code":    "internal_error",
				"message": "server borked",
			})
		}),
		provider,
		fastRetryPolicy(),
	)

	req := newRequest(t, "GET", srv.URL+"/test", nil)
	_, err := tr.RoundTrip(req)

	if !errors.Is(err, ic.ErrServer) {
		t.Errorf("expected ErrServer, got %v", err)
	}

	var se *ic.ServerError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ServerError, got %T", err)
	}
	if se.RequestID() != "int-exhaust" {
		t.Errorf("RequestID = %q, want %q", se.RequestID(), "int-exhaust")
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Should have retry.giving_up event.
	givingUp := 0
	for _, ev := range spans[0].Events {
		if ev.Name == "retry.giving_up" {
			givingUp++
		}
	}
	if givingUp != 1 {
		t.Errorf("retry.giving_up events = %d, want 1", givingUp)
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status = %v, want Error", spans[0].Status.Code)
	}
}

func TestIntegration_401ReFetchSuccess(t *testing.T) {
	t.Parallel()
	callNum := int32(0)
	provider := &countingProvider{tokens: []string{"old-tok", "new-tok"}}

	tr, srv, exp := newIntegrationTransport(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := atomic.AddInt32(&callNum, 1)
			if n == 1 {
				w.Header().Set("X-Request-Id", "int-401")
				w.WriteHeader(401)
				return
			}
			// Accept the new token.
			if r.Header.Get("Authorization") != "Bearer new-tok" {
				t.Errorf("expected new token, got %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("X-Request-Id", "int-401-ok")
			w.WriteHeader(200)
			w.Write([]byte(`{"authenticated": true}`))
		}),
		provider,
		noRetryPolicy(),
	)

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "GetMe",
		Resource:    "me",
	})
	req := newRequest(t, "GET", srv.URL+"/me", nil)
	req = req.WithContext(ctx)

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Provider should have been called 2 times.
	if n := provider.callCount(); n != 2 {
		t.Errorf("provider calls = %d, want 2", n)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
}

func TestIntegration_Redaction_SensitiveDataNotInSpans(t *testing.T) {
	t.Parallel()
	secretToken := "ULTRA_SECRET_TOKEN_NEVER_LEAK_THIS"
	secretAPIKey := "api_key_value_never_leak"
	provider := &countingProvider{tokens: []string{secretToken}}

	tr, srv, exp := newIntegrationTransport(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Request-Id", "int-redact")
			w.WriteHeader(200)
		}),
		provider,
		noRetryPolicy(),
	)

	req := newRequest(t, "GET", srv.URL+"/test?api_key="+secretAPIKey+"&page=1", nil)
	req.Header.Set("X-Api-Key", secretAPIKey)
	req.Header.Set("Cookie", "session=secret_session_value")

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Serialize all span attributes and events to a string for inspection.
	var sb strings.Builder
	for _, attr := range spans[0].Attributes {
		sb.WriteString(string(attr.Key))
		sb.WriteString("=")
		sb.WriteString(attr.Value.Emit())
		sb.WriteString("\n")
	}
	for _, ev := range spans[0].Events {
		sb.WriteString(ev.Name)
		sb.WriteString("\n")
		for _, attr := range ev.Attributes {
			sb.WriteString(string(attr.Key))
			sb.WriteString("=")
			sb.WriteString(attr.Value.Emit())
			sb.WriteString("\n")
		}
	}

	output := sb.String()

	// The bearer token must never appear.
	if strings.Contains(output, secretToken) {
		t.Error("bearer token appeared in span attributes/events")
	}

	// The API key from the query string should be redacted.
	if strings.Contains(output, secretAPIKey) {
		t.Error("API key appeared in span attributes/events")
	}

	// Verify the URL was redacted but still has the structure.
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "url.full" {
			urlVal := attr.Value.AsString()
			if strings.Contains(urlVal, secretAPIKey) {
				t.Errorf("url.full contains API key: %q", urlVal)
			}
			// The URL encoding may render [REDACTED] as %5BREDACTED%5D.
			if !strings.Contains(urlVal, "[REDACTED]") && !strings.Contains(urlVal, "REDACTED") {
				t.Errorf("url.full should contain REDACTED marker: %q", urlVal)
			}
		}
	}
}
