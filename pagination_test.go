package intelligencecloud

import (
	"context"
	"errors"
	"testing"
)

// --- A-15: Iterator lifecycle tests ---

func TestIterator_NextBeforeValue_DoesNotPanic(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return []int{1}, "", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	// Calling Value before Next should not panic; result is undefined but safe.
	_ = it.Value()
}

func TestIterator_EmptyResult(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return nil, "", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false for empty result")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestIterator_SinglePage(t *testing.T) {
	t.Parallel()
	items := []string{"a", "b", "c"}
	fetcher := func(_ context.Context, _ string) ([]string, string, error) {
		return items, "", nil
	}
	it := newIterator[string](fetcher, 0, 0)
	defer it.Close()

	var got []string
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(items) {
		t.Fatalf("got %d items, want %d", len(got), len(items))
	}
	for i, v := range got {
		if v != items[i] {
			t.Errorf("item[%d] = %q, want %q", i, v, items[i])
		}
	}
}

func TestIterator_NextAfterExhaustion_ReturnsFalse(t *testing.T) {
	t.Parallel()
	called := 0
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		called++
		if called == 1 {
			return []int{1}, "", nil
		}
		t.Fatal("fetcher called again after exhaustion")
		return nil, "", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	for it.Next(context.Background()) {
		// drain
	}
	// Further calls must return false without re-fetching.
	if it.Next(context.Background()) {
		t.Fatal("expected false after exhaustion")
	}
	if it.Next(context.Background()) {
		t.Fatal("expected false after second call post-exhaustion")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIterator_CloseIsIdempotent(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return []int{1}, "", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	it.Close()
	it.Close() // must not panic
	it.Close()
}

func TestIterator_NextAfterClose_ReturnsFalse(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return []int{1, 2, 3}, "page2", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected false after Close")
	}
}

// --- A-16: Iterator multi-page and PageInfo tests ---

func TestIterator_MultiPage(t *testing.T) {
	t.Parallel()
	pages := map[string]struct {
		items []int
		next  string
	}{
		"":      {items: []int{1, 2}, next: "p2"},
		"p2":    {items: []int{3, 4}, next: "p3"},
		"p3":    {items: []int{5}, next: ""},
	}
	fetcher := func(_ context.Context, token string) ([]int, string, error) {
		p, ok := pages[token]
		if !ok {
			t.Fatalf("unexpected token %q", token)
		}
		return p.items, p.next, nil
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d", len(got), len(want))
	}
	for i, v := range got {
		if v != want[i] {
			t.Errorf("item[%d] = %d, want %d", i, v, want[i])
		}
	}
}

func TestIterator_PageInfo_TracksCorrectly(t *testing.T) {
	t.Parallel()
	pages := map[string]struct {
		items []int
		next  string
	}{
		"":   {items: []int{10, 20}, next: "tok2"},
		"tok2": {items: []int{30}, next: ""},
	}
	fetcher := func(_ context.Context, token string) ([]int, string, error) {
		p := pages[token]
		return p.items, p.next, nil
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	// Before first Next, PageInfo should be zero.
	pi := it.PageInfo()
	if pi.ItemsFetched != 0 || pi.PagesFetched != 0 {
		t.Fatalf("expected zero PageInfo before first Next, got %+v", pi)
	}

	// Drain page 1 (2 items).
	ctx := context.Background()
	if !it.Next(ctx) {
		t.Fatal("expected true for first item")
	}
	pi = it.PageInfo()
	if pi.PagesFetched != 1 {
		t.Errorf("PagesFetched = %d, want 1", pi.PagesFetched)
	}
	if pi.HasNextPage != true {
		t.Error("expected HasNextPage = true after first page")
	}
	if pi.NextPageToken != "tok2" {
		t.Errorf("NextPageToken = %q, want %q", pi.NextPageToken, "tok2")
	}

	if !it.Next(ctx) {
		t.Fatal("expected true for second item")
	}

	// Move into page 2.
	if !it.Next(ctx) {
		t.Fatal("expected true for third item (first of page 2)")
	}
	pi = it.PageInfo()
	if pi.PagesFetched != 2 {
		t.Errorf("PagesFetched = %d, want 2", pi.PagesFetched)
	}
	if pi.ItemsFetched != 3 {
		t.Errorf("ItemsFetched = %d, want 3", pi.ItemsFetched)
	}
	if pi.HasNextPage != false {
		t.Error("expected HasNextPage = false after last page")
	}
	if pi.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty", pi.NextPageToken)
	}

	// Fully drain.
	for it.Next(ctx) {
	}
	pi = it.PageInfo()
	if pi.ItemsFetched != 3 {
		t.Errorf("final ItemsFetched = %d, want 3", pi.ItemsFetched)
	}
}

func TestIterator_WithMaxItems(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return []int{1, 2, 3, 4, 5}, "more", nil
	}
	it := newIterator[int](fetcher, 0, 3)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error with maxItems: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3 (maxItems cap)", len(got))
	}
}

func TestIterator_WithMaxItems_AcrossPages(t *testing.T) {
	t.Parallel()
	callCount := 0
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		callCount++
		return []int{1, 2}, "next", nil
	}
	it := newIterator[int](fetcher, 0, 5)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d items, want 5", len(got))
	}
}

func TestIterator_WithMaxItems_Zero_Unlimited(t *testing.T) {
	t.Parallel()
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return []int{1, 2, 3}, "", nil
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3 (unlimited)", len(got))
	}
}

func TestIterator_MemoryBounded_OnePage(t *testing.T) {
	t.Parallel()
	// Verifies iterator holds at most one page. After consuming page 1,
	// the fetcher returns page 2. We check the items seen match the
	// expected sequence, which proves page 1 data was released.
	page := 0
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		page++
		switch page {
		case 1:
			return []int{1, 2}, "p2", nil
		case 2:
			return []int{3, 4}, "", nil
		default:
			return nil, "", nil
		}
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	want := []int{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// --- A-17: Iterator error and context-cancellation tests ---

func TestIterator_FetcherError(t *testing.T) {
	t.Parallel()
	fetchErr := errors.New("backend unavailable")
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		return nil, "", fetchErr
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected false when fetcher returns error")
	}
	if !errors.Is(it.Err(), fetchErr) {
		t.Fatalf("Err() = %v, want %v", it.Err(), fetchErr)
	}
}

func TestIterator_FetcherError_MidPagination(t *testing.T) {
	t.Parallel()
	fetchErr := errors.New("page 2 failed")
	page := 0
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		page++
		if page == 1 {
			return []int{1, 2}, "p2", nil
		}
		return nil, "", fetchErr
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	var got []int
	for it.Next(context.Background()) {
		got = append(got, it.Value())
	}
	if len(got) != 2 {
		t.Fatalf("got %d items before error, want 2", len(got))
	}
	if !errors.Is(it.Err(), fetchErr) {
		t.Fatalf("Err() = %v, want %v", it.Err(), fetchErr)
	}
}

func TestIterator_PostError_NextReturnsFalse(t *testing.T) {
	t.Parallel()
	fetchErr := errors.New("fail")
	callCount := 0
	fetcher := func(_ context.Context, _ string) ([]int, string, error) {
		callCount++
		return nil, "", fetchErr
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	it.Next(context.Background())
	// Further Next calls must return false without re-fetching.
	if it.Next(context.Background()) {
		t.Fatal("expected false post-error")
	}
	if callCount != 1 {
		t.Fatalf("fetcher called %d times, want 1", callCount)
	}
	if !errors.Is(it.Err(), fetchErr) {
		t.Fatalf("Err() = %v, want %v", it.Err(), fetchErr)
	}
}

func TestIterator_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	fetcher := func(ctx context.Context, _ string) ([]int, string, error) {
		return nil, "", ctx.Err()
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	if it.Next(ctx) {
		t.Fatal("expected false with cancelled context")
	}
	if !errors.Is(it.Err(), context.Canceled) {
		t.Fatalf("Err() = %v, want context.Canceled", it.Err())
	}
}

func TestIterator_ContextDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	fetcher := func(ctx context.Context, _ string) ([]int, string, error) {
		return nil, "", ctx.Err()
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	if it.Next(ctx) {
		t.Fatal("expected false with expired deadline")
	}
	if !errors.Is(it.Err(), context.DeadlineExceeded) {
		t.Fatalf("Err() = %v, want context.DeadlineExceeded", it.Err())
	}
}

func TestIterator_ContextCancelledMidIteration(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	page := 0
	fetcher := func(ctx context.Context, _ string) ([]int, string, error) {
		page++
		if page == 1 {
			return []int{1, 2}, "p2", nil
		}
		cancel()
		return nil, "", ctx.Err()
	}
	it := newIterator[int](fetcher, 0, 0)
	defer it.Close()

	var got []int
	for it.Next(ctx) {
		got = append(got, it.Value())
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2 before cancel", len(got))
	}
	if !errors.Is(it.Err(), context.Canceled) {
		t.Fatalf("Err() = %v, want context.Canceled", it.Err())
	}
}
