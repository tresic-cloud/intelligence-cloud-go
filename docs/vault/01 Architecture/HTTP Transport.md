---
title: HTTP Transport
type: architecture
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - architecture
  - sdk/transport
  - go
  - http
aliases:
  - Transport
  - RoundTripper
  - Transport Pipeline
related:
  - "[[Overview]]"
  - "[[ADR-006 RoundTripper-based Retry with Full Jitter]]"
  - "[[Telemetry Contract]]"
  - "[[Retry Policy]]"
  - "[[Authentication]]"
  - "[[Security Posture]]"
---

# HTTP Transport

All outbound HTTP traffic from the SDK flows through a single `http.RoundTripper` implementation in `internal/transport/transport.go`. This is the standard Go idiom for composing cross-cutting HTTP concerns.

## Pipeline

Each request passes through these stages in order:

```
1. User-Agent injection
2. CredentialProvider.Token(ctx) -> Authorization header
3. W3C trace-context propagation (traceparent, tracestate, baggage)
4. OTel span start (intelligencecloud.{operationID})
5. Base transport round-trip (net/http default or user-supplied)
6. Capture X-Request-Id from response
7. Retry decision
8. On final response: map HTTP status to typed error
9. OTel span end with attributes
```

The transport wraps whatever base `http.RoundTripper` the user provides (or `http.DefaultTransport`). It does not replace it.

## Retry Behaviour

The transport retries requests that hit transient failures:

- **5xx responses** (500, 502, 503, 504)
- **429 Too Many Requests** (with `Retry-After` honouring)
- **Transport-level errors** (DNS, connection, TLS handshake)

Non-429 4xx responses are **never** retried. See [[Retry Policy]] for the full backoff math.

### 401 Re-fetch

On a 401 response, the transport invalidates the cached credential and re-invokes `CredentialProvider.Token(ctx)` exactly once. If the provider returns a fresh token, the request retries with the new token. If it fails again, a `*AuthenticationError` is surfaced. This bounds the re-fetch loop to a single additional attempt.

### Body Buffering

Request bodies are buffered before the first attempt so they can be replayed on retries. The buffer is bounded by the `MaxDelay` timeout and context deadline.

## Secret Redaction

The transport never leaks secrets into telemetry or logs:

- The `Authorization` header value never appears in OTel span attributes, events, or log records.
- URL query parameters whose names match `/token|secret|key|password/i` are replaced with `[REDACTED]` in the `url.full` span attribute.
- A redacting `slog.Handler` wrapper strips any log attribute whose key matches the same regex.
- Headers listed in `intelligencecloud.RedactedHeaders` (default: `Authorization`, `Cookie`, `X-Api-Key`, `X-Token`) are excluded from all telemetry.

See [[Security Posture]] for the broader threat model and [[Telemetry Contract#Prohibited content]] for the full exclusion list.

## OTel Span Emission

Each outbound request produces one OTel span with kind `SpanKindClient`, named `intelligencecloud.{operationID}`. The span carries HTTP semantic convention attributes (semconv v1.24) plus SDK-specific attributes. When no `TracerProvider` is configured, the no-op tracer produces zero-allocation spans. See [[Telemetry Contract]] for the complete attribute table.

### Span Events

During retry sequences, the transport emits span events:

| Event | When |
|---|---|
| `retry.attempt_failed` | After each failed attempt before sleeping |
| `retry.giving_up` | When retries are exhausted |

## Context Propagation

The transport injects W3C trace-context headers on every outbound request using the configured `propagation.TextMapPropagator` (default: W3C trace-context + baggage composite). If the caller's context already has an active span, the SDK propagates it without modification.

## Structured Logging

When a `*slog.Logger` is configured, the transport emits structured log records at these levels:

| Level | Event |
|---|---|
| DEBUG | Request start (operation, method, URL template, attempt number) |
| DEBUG | Response received (operation, status, request ID, duration) |
| WARN | Retry sleeping (operation, attempt, backoff, cause) |
| WARN | Backend returned error (operation, status, code, request ID) |
| ERROR | Transport error non-retryable or retries exhausted |

Every log record includes `trace_id` and `span_id` attributes when a span is active, enabling correlation between logs and traces.

## Testing

The transport is tested with:

- **Unit tests** against `httptest.Server` for each concern (auth injection, retry, telemetry, redaction, error mapping)
- **Integration tests** with an in-memory OTel exporter verifying end-to-end span correctness
- **Benchmarks** proving zero-cost no-op tracer guarantee (< 100 ns/op overhead, 0 allocations)
- **Race detection** (`-race`) under 100 concurrent goroutines
