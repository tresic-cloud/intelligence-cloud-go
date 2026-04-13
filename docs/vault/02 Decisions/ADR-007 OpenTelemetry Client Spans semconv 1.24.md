---
title: "ADR-007: OpenTelemetry Client Spans semconv 1.24"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-007
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Telemetry Contract]]"
  - "[[Overview]]"
tags:
  - decision/accepted
  - sdk/telemetry
  - opentelemetry
  - observability
aliases:
  - OTel ADR
  - Telemetry ADR
  - R5
  - R6
related:
  - "[[Telemetry Contract]]"
  - "[[HTTP Transport]]"
  - "[[Observability Runbook]]"
  - "[[Constitution Compliance]]"
---

# ADR-007: OpenTelemetry Client Spans semconv 1.24

## Status

Accepted

## Context

The project constitution requires observability for all production code. The constitution's reference pattern (Prometheus `InstrumentedService` / `InstrumentedRepository` decorators) is oriented toward backend HTTP handlers. For a client-side SDK, a different adaptation is needed.

The SDK must emit telemetry that integrates with consumers' existing observability stacks (Datadog, Honeycomb, Jaeger, etc.) without taking a hard runtime dependency on any specific tracer, exporter, or log sink. When no tracer is configured, the overhead must be zero.

## Decision

The SDK holds a `trace.TracerProvider` reference (default: no-op) injected via `WithTracerProvider(tp)`. The transport layer starts a span named `intelligencecloud.{operationID}` around every outbound request, applies the W3C trace-context propagator, and records attributes per OpenTelemetry HTTP client semantic conventions (semconv v1.24). SDK-specific attributes include `intelligencecloud.request_id`, `intelligencecloud.retry_attempt`, and `intelligencecloud.sdk_version`.

The observability adaptation for Constitution IV is:

- **Tracing**: OTel client spans (equivalent to trace-correlated logging)
- **Logging**: `slog.Logger` hook with `traceId` auto-injected from span context
- **Metrics**: deferred to consumer instrumentation via OTel metric API or user-supplied middleware (the SDK has no service boundary of its own)
- **Business error classification**: typed error hierarchy (see [[ADR-008 Typed Error Hierarchy with Sentinels]])

## Consequences

### Positive

- **Vendor-neutral**: the `trace` API is stable (1.0+). Consumers wire any exporter without SDK changes.
- **Auto-classification**: semconv compliance means Datadog (Tresic's APM backend) auto-classifies spans as HTTP client calls without custom processors.
- **Zero-cost no-op**: when no tracer provider is configured, span operations compile to near-zero overhead (benchmarked at < 100 ns/op, 0 allocations).
- **Correlation**: trace IDs flow through logs, spans, and error objects, enabling end-to-end debugging.

### Negative

- **No built-in metrics**: the SDK does not emit Prometheus metrics. Consumers who want request-rate or latency counters must instrument at their service level.
- **Semconv churn**: if OTel semconv changes, span attributes may need updating. Mitigated by pinning to semconv 1.24 and treating attribute changes as a minor-version-bump event.

## Alternatives Considered

### Embed otelhttp.NewTransport directly

Simpler but opinionated about span naming (`HTTP {method}`) and does not know operation IDs. The SDK wraps instead of delegates to provide richer span names and SDK-specific attributes.

### Datadog-native tracer (dd-trace-go)

Locks out other tracers and forces a runtime dependency on the large `dd-trace-go` module tree. Rejected per the specification's "no hard dependency" requirement.

### No tracing in v1

Rejected because observability is a constitution principle and retrofitting tracing into a released SDK would be a breaking change to transport internals.

## References

- [[Telemetry Contract]]
- [[Observability Runbook]]
- [[Constitution Compliance]]
- [OTel HTTP Client Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/http/http-spans/)
