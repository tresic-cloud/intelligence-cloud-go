---
title: "ADR-009: Generic Iterator T for Pagination"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-009
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Pagination]]"
  - "[[SDK Surface]]"
tags:
  - decision/accepted
  - sdk/pagination
  - go
aliases:
  - Pagination ADR
  - Iterator ADR
  - R8
related:
  - "[[Pagination]]"
  - "[[SDK Surface]]"
---

# ADR-009: Generic Iterator T for Pagination

## Status

Accepted

## Context

The specification requires that list endpoints not force callers to manually assemble next-page tokens, and that memory usage be bounded (no loading all pages into memory). Post-Go 1.18 generics make it possible to provide type-safe iteration without code generation for each resource type.

## Decision

Expose a generic `Iterator[T]` with the following API:

```go
type Iterator[T any] struct{ ... }
func (it *Iterator[T]) Next(ctx context.Context) bool
func (it *Iterator[T]) Value() T
func (it *Iterator[T]) Err() error
func (it *Iterator[T]) PageInfo() PageInfo
func (it *Iterator[T]) Close()
```

Each resource's `List*` method returns an `*Iterator[T]`. The iterator fetches pages on demand using the backend's page-token contract. Callers can cap results via `WithMaxItems(n)`.

## Consequences

### Positive

- **Type safety**: each iterator yields the correct resource type at compile time.
- **Idiomatic**: matches the pattern used by Google Cloud Go, Stripe Go, and DigitalOcean Godo.
- **Bounded memory**: only one page of items plus the next-page token is held in memory at a time.
- **Future-compatible**: maps cleanly to Go 1.23's `range-over-func` if the SDK later wants to expose that convenience.

### Negative

- **Learning curve**: consumers must understand the `Next(ctx)` / `Value()` / `Err()` pattern, though this is well-established in Go.
- **Single-use**: an iterator can only be traversed once. Callers who need to re-iterate must create a new one.

## Alternatives Considered

### Return ([]T, nextToken, error) tuples

Leaks pagination mechanics to the caller. The specification explicitly forbids requiring callers to manually reassemble pagination tokens.

### Channel-based streaming

Lifecycle and cancellation semantics are harder to get right. Channels do not compose as well with `context.Context` deadlines, and leaked goroutines are a common pitfall.

## References

- [[Pagination]]
- [[SDK Surface]]
