---
title: Pagination
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - sdk
  - pagination
  - iterator
aliases:
  - Iterator
  - Iterator[T]
  - Paginator
related:
  - "[[02 Decisions/ADR-009 Generic Iterator T for Pagination]]"
  - "[[01 Architecture/SDK Surface]]"
  - "[[Error Hierarchy]]"
  - "[[SDK Surface]]"
---

# Pagination

Every `List*` method in the SDK returns an `*Iterator[T]` -- a generic paginator that fetches pages on demand from the backend. The iterator abstracts away page tokens, page sizes, and backend pagination formats, presenting a simple sequential interface to the caller.

## Iterator[T] Surface

```go
type Iterator[T any] struct{ /* internal */ }

func (it *Iterator[T]) Next(ctx context.Context) bool  // Advances to the next item; fetches a new page if needed
func (it *Iterator[T]) Value() T                        // Returns the current item
func (it *Iterator[T]) Err() error                      // Returns the error that caused Next to return false, or nil
func (it *Iterator[T]) PageInfo() PageInfo               // Returns pagination metadata
func (it *Iterator[T]) Close()                           // Releases internal resources (idempotent)
```

## Invariants

- **`Next` before `Value`**: `Value()` is undefined until the first successful `Next(ctx)` call. Always call `Next` first.
- **Post-false is terminal**: once `Next` returns `false`, all subsequent calls to `Next` also return `false`. Check `Err()` to distinguish exhaustion (nil) from failure.
- **`Close` is idempotent**: calling `Close()` multiple times is safe. Use `defer it.Close()` immediately after obtaining the iterator.
- **Memory bounded to one page**: the iterator holds at most one page of items plus the next-page token in memory at any time. Consumed items are eligible for garbage collection as soon as `Next` advances past them.
- **Single-use**: an iterator can only be traversed once. To re-iterate, create a new iterator from the `List*` method.

## PageInfo

The `PageInfo()` method returns metadata about the pagination state:

```go
type PageInfo struct {
    ItemsFetched  int    // Total items yielded so far across all pages
    PagesFetched  int    // Number of pages fetched from the backend
    HasNextPage   bool   // Whether the backend indicates more pages exist
    NextPageToken string // Opaque cursor for the next page (empty if exhausted)
}
```

## Standard Consumer Loop

The idiomatic way to consume a paginated list:

```go
it := client.Resellers.List(ctx, ic.WithPageSize(50), ic.WithMaxItems(500))
defer it.Close()
for it.Next(ctx) {
    r := it.Value()
    fmt.Printf("Reseller: %s (%s)\n", r.Name, r.ID)
}
if err := it.Err(); err != nil {
    return fmt.Errorf("listing resellers: %w", err)
}
```

The loop fetches pages transparently as items are consumed. If `WithMaxItems(500)` is set, `Next` returns `false` with a nil `Err()` once 500 items have been yielded, even if more pages exist on the backend.

## List Options

These options are passed to `List*` methods and affect iterator behaviour:

| Option | Effect | Constraint |
|---|---|---|
| `WithPageSize(n)` | Hints the backend's page size (`page_size` query param) | `n > 0` |
| `WithMaxItems(n)` | Hard cap on total items returned; 0 = unlimited | `n >= 0` |
| `WithFilter(key, value)` | Passes a backend-defined filter parameter | Key/value are backend-specific |

Options are validated at construction time. `WithPageSize(0)` or `WithMaxItems(-1)` will cause the list call to fail.

## Backend Pagination Shapes

The Intelligence Cloud API uses two pagination response shapes depending on the endpoint:

1. **Flat cursor**: top-level `next_cursor` and `has_more` fields alongside the `data` array.
2. **Nested pagination**: a `pagination` object containing `next_cursor`, `has_more`, `total_count`.

The SDK smooths these into a single `Iterator[T]` interface. Consumers never see the underlying pagination format -- the iterator handles both transparently by detecting the response shape during deserialization.

## Memory Profile

The iterator is designed for bounded memory usage:

- At most **one page** of items is held in memory at a time.
- As `Next` advances through the current page, consumed items become eligible for garbage collection.
- When the current page is exhausted, the iterator fetches the next page and releases the previous page's buffer.
- `Close()` releases any remaining internal buffers immediately.

This makes iterators safe for very large result sets (thousands of items) without risk of unbounded memory growth.

## Error Handling

When a page fetch fails, `Next` returns `false` and `Err()` returns the error. The error is a typed SDK error (see [[Error Hierarchy]]) -- for example, a `*AuthenticationError` if the token expired mid-iteration, or a `*ServerError` if the backend returned a 500.

```go
it := client.Resellers.List(ctx, ic.WithPageSize(100))
defer it.Close()
for it.Next(ctx) {
    process(it.Value())
}
if err := it.Err(); err != nil {
    if errors.Is(err, ic.ErrAuthentication) {
        // token expired during pagination -- re-authenticate and restart
    }
    return err
}
```

## See Also

- [[02 Decisions/ADR-009 Generic Iterator T for Pagination]] -- the ADR justifying the `Iterator[T]` design
- [[01 Architecture/SDK Surface]] -- `List*` methods and list options in the public API
- [[Error Hierarchy]] -- typed errors that `Err()` may return
- [[Retry Policy]] -- how transient failures during page fetches are retried automatically
- [[Operation IDs]] -- which operations support pagination
