// Package intelligencecloud provides the Go SDK for the Intelligence Cloud
// platform. This file implements the generic Iterator[T] pagination helper
// used by all List* methods on resource services.
package intelligencecloud

import "context"

// PageInfo holds metadata about the pagination state of an Iterator.
type PageInfo struct {
	// ItemsFetched is the total number of items returned across all pages
	// consumed so far.
	ItemsFetched int

	// PagesFetched is the total number of backend page-fetch calls that
	// have completed so far.
	PagesFetched int

	// HasNextPage is true when the backend indicated that more results are
	// available beyond the most-recently fetched page.
	HasNextPage bool

	// NextPageToken is the opaque token that would be sent to the backend
	// to retrieve the next page. It is empty when HasNextPage is false.
	NextPageToken string
}

// pageFetcher is the function signature accepted by the iterator constructor.
// It fetches one page of results given a context and a page token (empty
// string for the first page). It returns the items in the page, the token
// for the next page (empty if no more pages), and any error.
type pageFetcher[T any] func(ctx context.Context, pageToken string) (items []T, nextToken string, err error)

// Iterator provides a sequential, demand-driven traversal over paginated
// backend results. It fetches at most one page of items at a time, keeping
// memory usage bounded.
//
// Usage:
//
//	it := client.Resellers.List(ctx, intelligencecloud.WithPageSize(50))
//	defer it.Close()
//	for it.Next(ctx) {
//	    r := it.Value()
//	    // ...
//	}
//	if err := it.Err(); err != nil {
//	    return err
//	}
type Iterator[T any] struct {
	fetch    pageFetcher[T]
	pageSize int
	maxItems int

	// Current page buffer and position within it.
	items []T
	pos   int

	// Pagination bookkeeping.
	nextToken    string
	pagesFetched int
	itemsYielded int
	hasNextPage  bool

	// Terminal state.
	err    error
	done   bool
	closed bool
	started bool
}

// newIterator constructs an Iterator that fetches pages using the provided
// fetcher function. pageSize is a hint for the backend (0 = default).
// maxItems caps the total items returned (0 = unlimited). This constructor
// is unexported — resource service List methods use it internally.
func newIterator[T any](fetch pageFetcher[T], pageSize, maxItems int) *Iterator[T] {
	return &Iterator[T]{
		fetch:    fetch,
		pageSize: pageSize,
		maxItems: maxItems,
	}
}

// Next advances the iterator to the next item. It returns true if an item
// is available via Value, or false when iteration is complete (either all
// items exhausted, maxItems reached, or an error occurred).
//
// Next fetches a new page from the backend when the current page buffer is
// exhausted and more pages are available. The provided context controls
// cancellation and deadlines for the page fetch.
//
// After Next returns false, further calls to Next return false without
// re-fetching.
func (it *Iterator[T]) Next(ctx context.Context) bool {
	if it.done || it.closed {
		return false
	}

	// Check maxItems cap.
	if it.maxItems > 0 && it.itemsYielded >= it.maxItems {
		it.done = true
		return false
	}

	// Advance position within current page buffer.
	if it.started {
		it.pos++
	}

	// Need to fetch a new page?
	if !it.started || it.pos >= len(it.items) {
		// Only fetch if this is the first call or there's a next page.
		if it.started && !it.hasNextPage {
			it.done = true
			return false
		}

		token := ""
		if it.started {
			token = it.nextToken
		}

		items, nextToken, err := it.fetch(ctx, token)
		if err != nil {
			it.err = err
			it.done = true
			return false
		}

		it.items = items
		it.pos = 0
		it.nextToken = nextToken
		it.hasNextPage = nextToken != ""
		it.pagesFetched++
		it.started = true

		if len(items) == 0 {
			it.done = true
			return false
		}
	}

	it.itemsYielded++
	return true
}

// Value returns the current item. It is undefined before the first call to
// Next returns true. After Next returns false, Value returns the last
// successfully yielded item (or the zero value if no items were yielded).
func (it *Iterator[T]) Value() T {
	if len(it.items) == 0 || it.pos >= len(it.items) {
		var zero T
		return zero
	}
	return it.items[it.pos]
}

// Err returns the error that caused Next to return false, or nil if
// iteration completed successfully (including when capped by maxItems).
func (it *Iterator[T]) Err() error {
	return it.err
}

// PageInfo returns a snapshot of the current pagination state.
func (it *Iterator[T]) PageInfo() PageInfo {
	return PageInfo{
		ItemsFetched:  it.itemsYielded,
		PagesFetched:  it.pagesFetched,
		HasNextPage:   it.hasNextPage,
		NextPageToken: it.nextToken,
	}
}

// Close releases resources held by the iterator. It is idempotent and safe
// to call from a deferred context. After Close, Next returns false.
func (it *Iterator[T]) Close() {
	it.closed = true
	it.done = true
	it.items = nil
}
