---
title: "ADR-001: OpenAPI Codegen with oapi-codegen"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-001
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Overview]]"
  - "[[SDK Surface]]"
tags:
  - decision/accepted
  - sdk
  - codegen
  - openapi
aliases:
  - OpenAPI Codegen ADR
  - R1
related:
  - "[[ADR-005 Pinned OpenAPI with Drift PRs]]"
  - "[[ADR-011 Hand-curated Generated Code Stop-gap]]"
  - "[[Operation IDs]]"
---

# ADR-001: OpenAPI Codegen with oapi-codegen

## Status

Accepted

## Context

The Intelligence Cloud platform exposes an HTTP API documented with an OpenAPI specification. The Go SDK must cover every operation in that spec with typed methods, request structs, and response structs. With approximately 21 operations today (and growing as the backend consolidates more per-feature contracts), manually transcribing these types is both error-prone and explicitly forbidden by the specification requirements. An automated code-generation approach is needed to keep the SDK in lockstep with the backend contract.

## Decision

Use `github.com/oapi-codegen/oapi-codegen/v2` in "types + low-level client" mode. Generate only request/response types and minimal client functions into `internal/generated/`. The public API surface is a hand-written layer on top that adds idiomatic Go options, pagination, typed errors, and context support.

The generated code lives behind the `internal/` boundary, so consumers never import it directly. This means we can change the generator tool later without breaking downstream code.

## Consequences

### Positive

- **Contract fidelity**: types are derived directly from the canonical OpenAPI, eliminating transcription drift.
- **Low maintenance burden**: adding a new backend endpoint requires only re-running codegen plus a thin wrapper.
- **Idiomatic output**: `oapi-codegen` produces clean Go code without reflection hacks or init-time registrations.
- **Alignment with upstream**: the intelligence-cloud backend (IDB-1354) names `oapi-codegen` as the smoke-test target for the consolidated spec.

### Negative

- **Generator coupling**: the project depends on `oapi-codegen`'s continued maintenance and its interpretation of OpenAPI semantics.
- **OpenAPI 3.1 gaps**: at the time of this decision, `oapi-codegen` has incomplete support for some OpenAPI 3.1 features, requiring a hand-curated stop-gap (see [[ADR-011 Hand-curated Generated Code Stop-gap]]).

## Alternatives Considered

### OpenAPITools/openapi-generator

A Java-based generator with broad language support. Rejected because: slower execution, less idiomatic Go output, heavier CI toolchain burden (requires JVM), and the generated surface tends to be more opinionated about HTTP client structure.

### ogen-go/ogen

A modern, fast, schema-first Go generator. Rejected because: its public API shape was still evolving at decision time, and its generated surface bleeds through more than `oapi-codegen`'s minimal output, making it harder to wrap with a stable hand-written layer.

### Hand-written client

Rejected because the specification explicitly requires that every operation be covered and that contract changes be reflected within one release cycle. Manual re-transcription of 21+ operations is the exact anti-pattern the requirements forbid.

## References

- [[Overview]]
- [[SDK Surface]]
- [[ADR-005 Pinned OpenAPI with Drift PRs]]
- [oapi-codegen on GitHub](https://github.com/oapi-codegen/oapi-codegen)
