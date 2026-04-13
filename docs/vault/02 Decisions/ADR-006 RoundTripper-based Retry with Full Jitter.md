---
title: "ADR-006: RoundTripper-based Retry with Full Jitter"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-006
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[HTTP Transport]]"
  - "[[Retry Policy]]"
tags:
  - decision/accepted
  - sdk/transport
  - retry
aliases:
  - Retry ADR
  - Full Jitter ADR
  - R4
related:
  - "[[HTTP Transport]]"
  - "[[Retry Policy]]"
  - "[[Error Hierarchy]]"
---

# ADR-006: RoundTripper-based Retry with Full Jitter

## Status

Accepted

## Context

The SDK must automatically retry transient failures (5xx, 429, transport errors) with exponential backoff. The retry logic must honour `Retry-After` headers, respect context deadlines, and never retry non-429 4xx responses. It must also compose cleanly with the rest of the HTTP pipeline (auth injection, telemetry, redaction).

## Decision

Implement retry as a custom `http.RoundTripper` layer in `internal/transport/transport.go` that wraps the user-supplied (or default) transport.

**Policy**: exponential backoff starting at 100 ms, doubling per attempt, capped at 30 s, with **full jitter** (uniform random between 0 and the computed backoff). Retry on 5xx, 429, and transport errors. Honour `Retry-After` (both integer-seconds and HTTP-date forms). Abort on context cancellation or deadline. Never retry non-429 4xx.

Default configuration: `MaxAttempts=4` (1 initial + 3 retries), `BaseDelay=100ms`, `MaxDelay=30s`, `Jitter=FullJitter`, `HonourRetryAfter=true`.

## Consequences

### Positive

- **Idiomatic Go**: `http.RoundTripper` is the canonical Go pattern for cross-cutting HTTP concerns. It composes cleanly with user-supplied transports and OTel instrumentation.
- **Minimises thundering herd**: full jitter (as opposed to equal jitter or no jitter) is proven in the AWS Architecture Blog's retry study to minimise correlated retry storms.
- **Single test surface**: retry logic is tested in one place rather than duplicated across 21+ resource methods.
- **Consumer control**: retry count, delays, and the master on/off switch are configurable via `RetryPolicy`.

### Negative

- **Body buffering**: request bodies must be buffered for replay on retries, adding memory overhead for large payloads.
- **Opacity**: consumers may not realise retries are happening unless they check logs or OTel spans. Mitigated by retry span events and WARN-level log records.

## Alternatives Considered

### hashicorp/go-retryablehttp

Full-featured, but pulls in an opinionated logger and clones request bodies. The SDK's needs are narrower and it needs precise control over body re-reads and telemetry.

### Retry at the SDK method layer

Wrapping each generated call individually with retry logic. Rejected because: bloats hand-written code surface (21+ methods would each carry identical retry scaffolding) and is harder to test comprehensively.

## References

- [[HTTP Transport]]
- [[Retry Policy]]
- [AWS Architecture Blog -- Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
