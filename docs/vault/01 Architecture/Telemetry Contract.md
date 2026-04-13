---
title: Telemetry Contract
type: architecture
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - architecture
  - sdk/telemetry
  - opentelemetry
  - slog
  - observability
aliases:
  - OTel Contract
  - Telemetry
  - Observability Contract
related:
  - "[[Overview]]"
  - "[[HTTP Transport]]"
  - "[[ADR-007 OpenTelemetry Client Spans semconv 1.24]]"
  - "[[Observability Runbook]]"
  - "[[Security Posture]]"
---

# Telemetry Contract

The SDK emits OpenTelemetry spans and structured `slog` log records for every outbound API call. This contract is stable across minor releases unless a backing OTel semantic convention changes.

## Zero-Cost Guarantee

When `WithTracerProvider` is not called, the SDK uses the OTel no-op tracer. `Tracer.Start(...)` returns the global no-op span, which performs no allocations for attributes, events, or span ends. A shipped benchmark (`BenchmarkClient_NoTracer`) enforces this stays under 100 ns/op additional overhead with 0 allocations.

Consumers who do not need tracing pay nothing for it.

## OTel Spans

### Naming

```
intelligencecloud.{operationID}
```

For example: `intelligencecloud.GetMe`, `intelligencecloud.ListResellers`.

### Span Kind

`trace.SpanKindClient` (always).

### Attributes

The SDK conforms to OpenTelemetry semantic conventions for HTTP client spans (semconv v1.24) and adds SDK-specific attributes:

| Attribute | Type | Source | Notes |
|---|---|---|---|
| `http.request.method` | string | request | `GET`, `POST`, etc. |
| `url.template` | string | generated code | Parameterised path, cardinality-safe (e.g. `/api/v1/resellers/{id}`) |
| `url.full` | string | request | Secrets redacted from query string |
| `server.address` | string | baseURL | Hostname |
| `server.port` | int | baseURL | Omitted if default for scheme |
| `network.protocol.name` | string | -- | Always `"http"` |
| `network.protocol.version` | string | response | e.g. `"1.1"`, `"2"` |
| `http.response.status_code` | int | response | Omitted on transport error |
| `http.request.body.size` | int | request | Bytes, when known |
| `http.response.body.size` | int | response | Bytes, when known |
| `user_agent.original` | string | request | SDK User-Agent string |
| `intelligencecloud.operation_id` | string | generated code | OpenAPI `operationId` |
| `intelligencecloud.resource` | string | generated code | Coarse resource area name |
| `intelligencecloud.request_id` | string | response header | From `X-Request-Id`, if present |
| `intelligencecloud.retry_attempt` | int | transport | `0` on first attempt |
| `intelligencecloud.retry_cause` | string | transport | On retried attempts only |
| `intelligencecloud.sdk_version` | string | build info | Semver (e.g. `v0.2.1`) |
| `error.type` | string | SDK | Concrete error type on failure |

### Span Status

| Outcome | Status | Description |
|---|---|---|
| 2xx / 3xx | `Unset` | Default |
| 4xx | `Unset` | Expected client error per semconv |
| 5xx | `Error` | Concrete error type |
| Transport error | `Error` | Wrapped error message |
| Context cancelled | `Error` | `"context cancelled"` |
| Context deadline | `Error` | `"context deadline exceeded"` |

### Span Events

| Event name | Attributes | When |
|---|---|---|
| `retry.attempt_failed` | `retry_attempt`, `backoff_ms`, `cause`, `retry_after_ms` (optional) | After each failed attempt |
| `retry.giving_up` | `total_attempts`, `last_cause` | Retries exhausted |

## Context Propagation

Outbound requests inject headers via the configured `propagation.TextMapPropagator`. The default is the composite W3C trace-context + baggage propagator. Every request carries `traceparent`, and optionally `tracestate` and `baggage` when present in the span context.

The SDK never forges or modifies `traceparent` if the caller's context already has an active span.

## Structured Logging

Consumer-supplied `*slog.Logger` (via `WithLogger`) receives these records:

| Level | Event | Key attributes |
|---|---|---|
| DEBUG | Request start | `operation_id`, `method`, `url.template`, `attempt` |
| DEBUG | Response received | `operation_id`, `status`, `request_id`, `duration_ms` |
| WARN | Retry sleeping | `operation_id`, `attempt`, `backoff_ms`, `cause` |
| WARN | Backend error | `operation_id`, `status`, `code`, `request_id` |
| ERROR | Transport error (non-retryable or exhausted) | `operation_id`, `error`, `attempts` |

### Trace-Log Correlation

Every log record includes `trace_id` and `span_id` attributes when a span is active. This enables correlation between structured logs and distributed traces in backends like Datadog.

### Redaction

The SDK wraps the consumer-supplied logger in a redacting handler that strips any attribute whose key matches `/token|secret|key|password/i`. The `Authorization` header is never logged.

## Prohibited Content

The following must never appear in any span attribute, event, or log record:

- Bearer tokens, refresh tokens, API keys (any form)
- Full request or response bodies (metadata like `body.size` is allowed)
- `Authorization` header value
- Cookie values
- Any header named in `intelligencecloud.RedactedHeaders`

This is enforced by a unit test that fires a request with sensitive headers through an in-memory exporter and asserts none of the sensitive material appears in captured telemetry.

See [[Security Posture]] for the broader security model.
