---
title: Observability Runbook
type: operational
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - ops
  - observability
  - opentelemetry
  - debugging
aliases:
  - OTel Runbook
  - Debugging Guide
  - Telemetry Runbook
related:
  - "[[01 Architecture/Telemetry Contract]]"
  - "[[01 Architecture/HTTP Transport]]"
  - "[[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24]]"
  - "[[Security Posture]]"
  - "[[Compliance Controls]]"
---

# Observability Runbook

This runbook explains how to read the telemetry the Intelligence Cloud Go SDK emits when debugging a consumer service. The SDK produces OpenTelemetry spans and structured `slog` log records for every outbound API call. When no tracer or logger is configured, the SDK is silent and allocates nothing for telemetry.

## Span Name Pattern

Every outbound API request produces one OTel span with kind `SpanKindClient`, named:

```
intelligencecloud.{operationID}
```

The `operationID` is the OpenAPI `operationId` from the backend specification. Examples:

- `intelligencecloud.GetMe`
- `intelligencecloud.ListResellers`
- `intelligencecloud.DeleteCompany`

Use this prefix to filter SDK spans in your APM tool's trace view.

## Key Attributes to Filter On

When investigating SDK behaviour in Datadog, Honeycomb, Jaeger, or any OTel-compatible APM backend, filter on these attributes:

| Attribute | Type | Use case |
|---|---|---|
| `intelligencecloud.operation_id` | string | Narrow to a specific API operation (e.g. `GetMe`, `ListResellers`) |
| `intelligencecloud.request_id` | string | Correlate an SDK span to a specific backend request for support triage |
| `http.response.status_code` | int | Filter by HTTP status (e.g. `429`, `500`, `401`) |
| `intelligencecloud.retry_attempt` | int | Find retried requests (`> 0` means at least one retry occurred) |
| `intelligencecloud.sdk_version` | string | Identify which SDK version emitted the span (useful during rollouts) |
| `error.type` | string | Filter by concrete error type (e.g. `RateLimitError`, `AuthenticationError`) |
| `url.template` | string | Group by endpoint path template (cardinality-safe, e.g. `/api/v1/resellers/{id}`) |
| `intelligencecloud.resource` | string | Filter by coarse resource area (`resellers`, `companies`, etc.) |

See [[01 Architecture/Telemetry Contract]] for the complete attribute table including HTTP semantic convention attributes.

## Span Events

During retry sequences, the SDK emits span events that provide detail about why retries occurred and when they stopped.

### `retry.attempt_failed`

Emitted after each failed attempt before the transport sleeps for the backoff interval.

| Attribute | Type | Description |
|---|---|---|
| `retry_attempt` | int | Zero-indexed attempt number |
| `backoff_ms` | int | Milliseconds the transport will sleep before the next attempt |
| `cause` | string | Why the attempt failed (e.g. `status_429`, `status_500`, `transport_error`) |
| `retry_after_ms` | int | Milliseconds from the `Retry-After` header, if present (optional) |

### `retry.giving_up`

Emitted when retries are exhausted and the transport surfaces the final error.

| Attribute | Type | Description |
|---|---|---|
| `total_attempts` | int | Total number of attempts made |
| `last_cause` | string | The cause of the final failed attempt |

In Datadog, expand the "Events" tab on the span detail view. In Jaeger, events appear in the span's "Logs" section.

## Log-to-Trace Correlation

Every log record the SDK emits includes `trace_id` and `span_id` attributes when a span is active in the calling context. This enables correlation between structured logs and distributed traces in backends like Datadog, where you can jump from a log line to the associated trace.

The correlation chain:

```
slog record (trace_id, span_id)
    |
    +---> OTel span (same trace_id, span_id)
              |
              +---> intelligencecloud.request_id (X-Request-Id from backend)
```

To correlate a user-reported error to the backend:

1. Find the `request_id` in the error object or log record.
2. Search your APM backend for `intelligencecloud.request_id = "<value>"`.
3. The matching span shows the full request lifecycle including retry attempts.

## Common Symptoms and Likely Causes

### 401s in a loop

**Symptom**: Your service logs show repeated `AuthenticationError` responses, or your APM shows spans with `http.response.status_code = 401` and `error.type = AuthenticationError`.

**Likely cause**: The `CredentialProvider` is returning the same expired token on each invocation. The SDK bounds the re-fetch cycle to exactly one retry: on a 401, it invalidates the cached credential, calls `CredentialProvider.Token(ctx)` once more, and retries the request. If the second attempt also returns 401, the SDK surfaces an `*AuthenticationError` and stops.

**What to check**: Verify that your `CredentialProvider` implementation actually fetches a fresh token when called (not returning a stale cached value from an outer layer). If using `RefreshFunc`, confirm that the underlying callback returns a token with a future `ExpiresAt` value.

See [[05 Reference/Authentication]] for the credential provider contract.

### 429s (rate limiting)

**Symptom**: Spans show `http.response.status_code = 429` and `retry.attempt_failed` events with `cause = status_429`.

**Likely cause**: The backend is rate-limiting your requests. The SDK automatically retries with exponential backoff and honours the `Retry-After` header.

**What to check**: Look at the `retry_after_ms` attribute on `retry.attempt_failed` events to see how long the backend is asking you to wait. If retries are exhausted, a `*RateLimitError` is surfaced. Consider reducing request concurrency or implementing client-side rate limiting.

See [[05 Reference/Retry Policy]] for the full backoff formula.

### Missing spans

**Symptom**: Your APM backend shows no `intelligencecloud.*` spans even though SDK calls are executing.

**Likely cause**: No `TracerProvider` was configured on the client. By default, the SDK uses the OTel no-op tracer, which emits nothing.

**Fix**: Pass your application's tracer provider when constructing the client:

```go
client := intelligencecloud.NewClient(
    intelligencecloud.WithCredentialProvider(creds),
    intelligencecloud.WithTracerProvider(otel.GetTracerProvider()),
)
```

If you are using the global tracer provider registered via `otel.SetTracerProvider(...)`, the call above wires it in. If you use a non-global provider, pass it directly.

### Missing log correlation

**Symptom**: SDK log records do not contain `trace_id` or `span_id` attributes, so you cannot jump from logs to traces.

**Likely cause**: No logger was supplied to the client. The default is a no-op `slog.Handler`.

**Fix**: Supply a logger via `WithLogger`:

```go
client := intelligencecloud.NewClient(
    intelligencecloud.WithCredentialProvider(creds),
    intelligencecloud.WithTracerProvider(otel.GetTracerProvider()),
    intelligencecloud.WithLogger(slog.Default()),
)
```

The SDK wraps the supplied logger in a `RedactingHandler` automatically -- you do not need to add redaction yourself.

## Verbose Logging

To enable verbose SDK logging, pass a `slog.Logger` configured at `DEBUG` level:

```go
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)

client := intelligencecloud.NewClient(
    intelligencecloud.WithCredentialProvider(creds),
    intelligencecloud.WithLogger(logger),
)
```

At `DEBUG` level, the SDK emits:

- **Request start** -- operation ID, HTTP method, URL template, attempt number
- **Response received** -- operation ID, status code, request ID, duration in milliseconds

At `WARN` level (also visible at DEBUG):

- **Retry sleeping** -- operation ID, attempt number, backoff duration, cause
- **Backend error** -- operation ID, status code, error code, request ID

At `ERROR` level:

- **Transport error** -- non-retryable failure or retries exhausted, with total attempt count

See [[01 Architecture/Telemetry Contract]] for the full log record schema.

## Performance Considerations

Benchmark data from the project's test suite (task C-13):

| Configuration | Allocations per call | Overhead per call |
|---|---|---|
| No tracer (no-op path) | 0 additional | < 100 ns |
| WithTracerProvider (active tracing) | ~14 additional | ~1.7 us |

The no-op path is verified by `BenchmarkClient_NoTracer` to remain at zero allocations. Active tracing adds modest overhead (~14 allocations, ~1.7 microseconds) that is negligible for typical API call patterns but may be noticeable in tight loops issuing thousands of requests per second. If your service makes SDK calls in a hot loop, consider whether tracing is needed for every call or whether sampling at the tracer-provider level is sufficient.

## See Also

- [[01 Architecture/Telemetry Contract]] -- complete span attribute table, span status rules, and prohibited content
- [[01 Architecture/HTTP Transport]] -- the transport pipeline that emits spans and logs
- [[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24]] -- decision record for the OTel integration approach
- [[05 Reference/Authentication]] -- CredentialProvider interface and token caching
- [[05 Reference/Retry Policy]] -- backoff formula, jitter strategy, and Retry-After parsing
- [[Security Posture]] -- redaction rules that affect what appears in telemetry
