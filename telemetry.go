package intelligencecloud

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// spanPrefix is the constant prefix for all SDK span names.
const spanPrefix = "intelligencecloud."

// SpanName returns the canonical OTel span name for the given OpenAPI
// operationID. The result is "intelligencecloud." followed by the
// operationID, e.g. "intelligencecloud.GetMe".
func SpanName(operationID string) string {
	return spanPrefix + operationID
}

// sensitiveKeyPattern matches query-parameter or attribute keys that
// contain sensitive material. Used by RedactURL and RedactingHandler.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)(token|secret|key|password)`)

// RedactURL returns the string representation of u with the values of
// query parameters whose key matches the sensitive-key pattern replaced
// by "[REDACTED]". The URL structure (scheme, host, path) is preserved.
func RedactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	redacted := false
	for k, vals := range q {
		if sensitiveKeyPattern.MatchString(k) {
			for i := range vals {
				vals[i] = "[REDACTED]"
			}
			q[k] = vals
			redacted = true
		}
	}
	if !redacted {
		return u.String()
	}
	cp := *u
	cp.RawQuery = q.Encode()
	return cp.String()
}

// RedactedHeaders is the default set of HTTP header names whose values
// must never appear in spans, logs, or error payloads. Callers may
// extend this slice before constructing a Client.
var RedactedHeaders = []string{
	"Authorization",
	"Cookie",
	"X-Api-Key",
	"X-Token",
}

// RedactHeaders returns a copy of h containing only headers that are NOT
// in the RedactedHeaders set. Header names are compared case-insensitively.
func RedactHeaders(h http.Header) map[string]string {
	blocked := make(map[string]struct{}, len(RedactedHeaders))
	for _, name := range RedactedHeaders {
		blocked[strings.ToLower(name)] = struct{}{}
	}

	out := make(map[string]string, len(h))
	for name, vals := range h {
		if _, ok := blocked[strings.ToLower(name)]; ok {
			continue
		}
		out[name] = strings.Join(vals, ", ")
	}
	return out
}

// Span attribute helpers

// HTTPRequestAttrs returns OTel attributes for an outbound HTTP request.
func HTTPRequestAttrs(method, urlTemplate, fullURL, userAgent string, bodySize int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", method),
		attribute.String("url.template", urlTemplate),
		attribute.String("url.full", fullURL),
		attribute.String("user_agent.original", userAgent),
	}
	if bodySize >= 0 {
		attrs = append(attrs, attribute.Int("http.request.body.size", bodySize))
	}
	return attrs
}

// HTTPResponseAttrs returns OTel attributes for an HTTP response.
func HTTPResponseAttrs(statusCode, bodySize int, protoVersion string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Int("http.response.status_code", statusCode),
		attribute.String("network.protocol.version", protoVersion),
	}
	if bodySize >= 0 {
		attrs = append(attrs, attribute.Int("http.response.body.size", bodySize))
	}
	return attrs
}

// ServerAttrs returns OTel attributes for the target server.
func ServerAttrs(host string, port int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("server.address", host),
		attribute.String("network.protocol.name", "http"),
	}
	if port > 0 {
		attrs = append(attrs, attribute.Int("server.port", port))
	}
	return attrs
}

// SDKAttrs returns OTel attributes for SDK-specific metadata.
func SDKAttrs(operationID, resource, requestID, sdkVersion string, retryAttempt int, retryCause string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("intelligencecloud.operation_id", operationID),
		attribute.String("intelligencecloud.resource", resource),
		attribute.String("intelligencecloud.sdk_version", sdkVersion),
		attribute.Int("intelligencecloud.retry_attempt", retryAttempt),
	}
	if requestID != "" {
		attrs = append(attrs, attribute.String("intelligencecloud.request_id", requestID))
	}
	if retryCause != "" {
		attrs = append(attrs, attribute.String("intelligencecloud.retry_cause", retryCause))
	}
	return attrs
}

// RedactingHandler is an slog.Handler wrapper that:
//   - Strips any attribute whose key matches the sensitive-key pattern
//     (token, secret, key, password — case-insensitive substring match).
//   - Injects trace_id and span_id attributes when a span is active in
//     the context.
//   - Delegates all other behaviour to the underlying handler.
type RedactingHandler struct {
	inner slog.Handler
}

// NewRedactingHandler wraps inner with sensitive-attribute stripping and
// trace-context injection.
func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

// Enabled delegates to the inner handler.
func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle filters sensitive attributes and injects trace context before
// delegating to the inner handler.
func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Build a new record with filtered attributes.
	filtered := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	r.Attrs(func(a slog.Attr) bool {
		if sensitiveKeyPattern.MatchString(a.Key) {
			return true // skip this attr
		}
		filtered.AddAttrs(a)
		return true
	})

	// Inject trace context if a span is active.
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		filtered.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.inner.Handle(ctx, filtered)
}

// WithAttrs returns a new handler with the given pre-filtered attributes.
func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var filtered []slog.Attr
	for _, a := range attrs {
		if sensitiveKeyPattern.MatchString(a.Key) {
			continue
		}
		filtered = append(filtered, a)
	}
	return &RedactingHandler{inner: h.inner.WithAttrs(filtered)}
}

// WithGroup returns a new handler with the given group name.
func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name)}
}
