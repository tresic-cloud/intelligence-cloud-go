---
title: "ADR-008: Typed Error Hierarchy with Sentinels"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-008
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Error Hierarchy]]"
  - "[[SDK Surface]]"
tags:
  - decision/accepted
  - sdk/errors
  - go
aliases:
  - Error ADR
  - Error Hierarchy ADR
  - R7
related:
  - "[[Error Hierarchy]]"
  - "[[SDK Surface]]"
  - "[[HTTP Transport]]"
---

# ADR-008: Typed Error Hierarchy with Sentinels

## Status

Accepted

## Context

The specification requires that every API error be classifiable by kind (authentication, authorization, validation, not-found, rate-limit, server) using only the public error API. Go developers expect to use `errors.Is` for quick classification and `errors.As` for detailed inspection. The error objects must also carry support-critical metadata: HTTP status, backend error code, human-readable message, request identifier, and originating operation.

## Decision

Define a public error hierarchy with:

- An `APIError` interface (embedding `error`) with methods `Status() int`, `Code() string`, `RequestID() string`, `Operation() string`.
- Eight concrete error types: `AuthenticationError` (401), `AuthorizationError` (403), `ValidationError` (400/422), `NotFoundError` (404), `ConflictError` (409), `RateLimitError` (429), `ServerError` (5xx), `UnexpectedError` (everything else).
- Seven sentinel errors (`ErrAuthentication`, `ErrAuthorization`, `ErrValidation`, `ErrNotFound`, `ErrConflict`, `ErrRateLimit`, `ErrServer`) enabling `errors.Is` matching.
- Each concrete type implements `Unwrap()` returning its sentinel, so `errors.Is(err, ErrNotFound)` works out of the box.
- `UnexpectedError` carries a bounded `CauseBody` (capped at 4 KiB) for debugging schema drift or unknown responses.

A separate `ConfigurationError` (returned only from `NewClient`) implements `error` but not `APIError`, distinguishing construction failures from API call failures.

## Consequences

### Positive

- **Dual access patterns**: `errors.Is` for quick checks, `errors.As` for detailed inspection. Both are idiomatic Go.
- **Support triage**: every error carries a `RequestID` that maps to backend logs.
- **Bounded memory**: `UnexpectedError.CauseBody` is capped at 4 KiB to prevent unbounded memory on large error responses.
- **Extensibility**: new error types and sentinels can be added in minor releases without breaking existing `default` branches.

### Negative

- **Surface area**: eight concrete types is a wide hierarchy. Mitigated by the sentinel layer, which lets most consumers handle only 2-3 cases.
- **Maintenance**: each new backend error category may require a new type. Mitigated by `UnexpectedError` as the catch-all.

## Alternatives Considered

### Single Error struct with Kind enum

Simpler, but does not compose with `errors.Is` in the ergonomic way Go consumers expect. A `switch err.Kind` pattern is less idiomatic than `errors.Is`.

### Codes-only (error with a Code() method)

Insufficient for compile-time exhaustiveness checks. Go switch-on-type gives better guarantees than string comparisons.

## References

- [[Error Hierarchy]]
- [[SDK Surface]]
