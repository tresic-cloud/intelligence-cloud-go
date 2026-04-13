# Contract: Telemetry (OTel Spans + Structured Logs)

**Scope**: the observability surface the SDK emits. Stable across minor releases unless a backing OTel semantic convention changes.

## OTel spans

### Span naming

```
intelligencecloud.{operationID}
```

where `{operationID}` is the OpenAPI `operationId` as generated — e.g. `intelligencecloud.GetMe`, `intelligencecloud.ListResellers`.

### Span kind

`trace.SpanKindClient`

### Attributes

Conforms to OpenTelemetry semantic conventions for HTTP client spans (semconv v1.24) PLUS the SDK-specific attributes listed below.

| Attribute | Type | Source | Notes |
|---|---|---|---|
| `http.request.method` | string | request | e.g. `"GET"` |
| `url.template` | string | generated code | parameterised path, cardinality-safe — e.g. `/api/v1/resellers/{id}` |
| `url.full` | string | request | **secrets redacted** — query-string keys matching `/token\|secret\|key\|password/i` are replaced with `[REDACTED]` |
| `server.address` | string | baseURL host | |
| `server.port` | int | baseURL port | omitted if default for scheme |
| `network.protocol.name` | string | `"http"` | always |
| `network.protocol.version` | string | response | e.g. `"1.1"`, `"2"` |
| `http.response.status_code` | int | response | omitted on transport error |
| `http.request.body.size` | int | request | bytes, when known |
| `http.response.body.size` | int | response | bytes, when known |
| `user_agent.original` | string | request | SDK UA string |
| `intelligencecloud.operation_id` | string | generated code | the OpenAPI `operationId` |
| `intelligencecloud.resource` | string | generated code | coarse-grained resource area (`resellers`, `companies`, …) |
| `intelligencecloud.request_id` | string | `X-Request-Id` response header | set only if response header present |
| `intelligencecloud.retry_attempt` | int | transport | `0` on first attempt |
| `intelligencecloud.retry_cause` | string | transport | on retried attempts only — `"status_429"`, `"status_500"`, …, `"transport_error"` |
| `intelligencecloud.sdk_version` | string | build info | semver, e.g. `"v0.2.1"` |
| `error.type` | string | SDK | concrete error type name on failure (e.g. `"RateLimitError"`) |

### Span status

| Outcome | Status | Description |
|---|---|---|
| 2xx / 3xx | `Unset` | (default) |
| 4xx | `Unset` | per semconv — 4xx is "expected client error" from the span's perspective |
| 5xx | `Error` | description = concrete error type |
| transport error | `Error` | description = wrapped error message |
| ctx cancelled | `Error` | description = `"context cancelled"` |
| ctx deadline | `Error` | description = `"context deadline exceeded"` |

### Span events

Emitted via `span.AddEvent` during retry sequences:

| Event name | Attributes |
|---|---|
| `retry.attempt_failed` | `retry_attempt` (int), `backoff_ms` (int), `cause` (string), `retry_after_ms` (int, optional) |
| `retry.giving_up` | `total_attempts` (int), `last_cause` (string) |

### Zero-cost no-op contract

When `WithTracerProvider` is not called, the client uses the OTel no-op tracer. Every `Tracer.Start(...)` returns the global no-op span implementation, which performs no allocations for attributes, events, or span ends. A benchmark `BenchmarkClient_NoTracer_vs_WithTracer` is shipped to enforce this ceiling remains under 100 ns/op additional overhead.

## Context propagation

Outbound requests inject headers via the configured `propagation.TextMapPropagator`. Default is the composite W3C trace-context + baggage propagator.

Concretely, every request carries (when a span is active):
- `traceparent`
- `tracestate` (when present in span context)
- `baggage` (when present in context)

The SDK never forges or modifies `traceparent` if the caller's context already has a span — it just propagates.

## Logging (`*slog.Logger`)

Consumer supplies a logger via `WithLogger`. Default: a no-op handler. The SDK emits:

| Level | Event | Attributes |
|---|---|---|
| DEBUG | request start | `operation_id`, `method`, `url.template`, `attempt` |
| DEBUG | response received | `operation_id`, `status`, `request_id`, `duration_ms` |
| WARN | retry sleeping | `operation_id`, `attempt`, `backoff_ms`, `cause` |
| WARN | backend returned error | `operation_id`, `status`, `code`, `request_id` |
| ERROR | transport error (non-retryable or retries exhausted) | `operation_id`, `error`, `attempts` |

**Trace-log correlation**: every emitted log record includes `trace_id` and `span_id` attributes when a span is active in the context (pulled from the span context at log-call time).

**Redaction**: the SDK wraps the consumer-supplied logger in a redacting handler that strips any attribute whose key matches `/token\|secret\|key\|password/i`. The `Authorization` header is never logged.

## Prohibited content

The following MUST NOT appear in any span attribute, event, or log record:

- Bearer tokens, refresh tokens, API keys (any form)
- Full request bodies (bounded metadata only; `http.request.body.size` is allowed, body content is not)
- Response bodies from non-error responses
- `Authorization` header value
- Cookie values
- Any header named in `intelligencecloud.RedactedHeaders` (default: `Authorization`, `Cookie`, `X-Api-Key`, `X-Token`)

Enforced by a unit test that fires a request with sensitive headers + a token through an in-memory exporter and asserts none of the sensitive material appears in captured telemetry.
