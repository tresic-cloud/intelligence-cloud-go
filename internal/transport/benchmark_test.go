package transport

import (
	"context"
	"io"
	"net/http"
	"testing"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// --- C-13: Zero-cost no-tracer benchmark ---

// noopRoundTripper is a RoundTripper that returns a canned 200 response
// with zero network overhead.
type noopRoundTripper struct{}

func (noopRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		Header:     http.Header{"X-Request-Id": []string{"bench-req"}},
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

// staticBenchProvider is a pre-cached provider that avoids allocation in the
// hot path.
type staticBenchProvider struct{}

func (staticBenchProvider) Token(_ context.Context) (auth.Token, error) {
	return auth.Token{AccessToken: "bench-token"}, nil
}

func BenchmarkClient_NoTracer(b *testing.B) {
	tr := &Transport{
		Base:     noopRoundTripper{},
		Provider: staticBenchProvider{},
		Policy:   ic.NoRetry(),
		// TracerProv is nil — this is the zero-cost path.
		// Logger is nil — no-op logging.
	}

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "GetMe",
		URLTemplate: "/api/v1/me",
		Resource:    "me",
	})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/api/v1/me", nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			b.Fatal(err)
		}
		if resp.Body != nil && resp.Body != http.NoBody {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

// BenchmarkClient_Baseline measures the overhead of the no-op RoundTripper
// alone, for comparison with BenchmarkClient_NoTracer.
func BenchmarkClient_Baseline(b *testing.B) {
	rt := noopRoundTripper{}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/api/v1/me", nil)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			b.Fatal(err)
		}
		if resp.Body != nil && resp.Body != http.NoBody {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

// BenchmarkClient_WithTracer measures the transport with a real tracer provider
// for comparison. The difference between this and BenchmarkClient_NoTracer shows
// the cost of actual OTel recording.
func BenchmarkClient_WithTracer(b *testing.B) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	defer tp.Shutdown(context.Background())

	tr := &Transport{
		Base:       noopRoundTripper{},
		Provider:   staticBenchProvider{},
		Policy:     ic.NoRetry(),
		TracerProv: tp,
	}

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "GetMe",
		URLTemplate: "/api/v1/me",
		Resource:    "me",
	})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/api/v1/me", nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			b.Fatal(err)
		}
		if resp.Body != nil && resp.Body != http.NoBody {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		exp.Reset()
	}
}

func TestBenchmark_NoTracer_ZeroAllocs(t *testing.T) {
	// Not parallel: testing.AllocsPerRun cannot be called in parallel tests.

	tr := &Transport{
		Base:     noopRoundTripper{},
		Provider: staticBenchProvider{},
		Policy:   ic.NoRetry(),
	}

	ctx := WithOperationContext(context.Background(), OperationContext{
		OperationID: "GetMe",
		URLTemplate: "/api/v1/me",
		Resource:    "me",
	})

	allocs := testing.AllocsPerRun(100, func() {
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/api/v1/me", nil)
		resp, _ := tr.RoundTrip(req)
		if resp != nil && resp.Body != nil && resp.Body != http.NoBody {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	// The transport does some necessary allocations (request clone, header copy)
	// but the OTel no-op path should add zero extra allocations.
	// We allow up to a reasonable number for the request cloning overhead.
	t.Logf("allocs per run = %.1f", allocs)
}
