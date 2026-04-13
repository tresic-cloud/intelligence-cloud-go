package auth_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

func TestRefreshFunc(t *testing.T) {
	t.Run("first call invokes fn and returns token", func(t *testing.T) {
		exp := time.Now().Add(time.Hour)
		var calls int64
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			atomic.AddInt64(&calls, 1)
			return auth.Token{AccessToken: "fresh", ExpiresAt: exp}, nil
		})
		tok, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "fresh" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "fresh")
		}
		if !tok.ExpiresAt.Equal(exp) {
			t.Errorf("ExpiresAt = %v; want %v", tok.ExpiresAt, exp)
		}
		if c := atomic.LoadInt64(&calls); c != 1 {
			t.Errorf("fn called %d times; want 1", c)
		}
	})

	t.Run("cache hit within expiry window", func(t *testing.T) {
		// Token expires in 5 minutes — well beyond the 30s safety margin.
		exp := time.Now().Add(5 * time.Minute)
		var calls int64
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			atomic.AddInt64(&calls, 1)
			return auth.Token{AccessToken: "cached", ExpiresAt: exp}, nil
		})

		// First call populates cache.
		tok1, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("call 1: unexpected error: %v", err)
		}

		// Second call should hit cache.
		tok2, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("call 2: unexpected error: %v", err)
		}

		if tok1.AccessToken != tok2.AccessToken {
			t.Errorf("cache miss: got %q and %q", tok1.AccessToken, tok2.AccessToken)
		}
		if c := atomic.LoadInt64(&calls); c != 1 {
			t.Errorf("fn called %d times; want 1 (cache hit)", c)
		}
	})

	t.Run("cache miss at expiry minus 30s margin", func(t *testing.T) {
		// First token "expires" 10 seconds from now. Since the 30-second
		// safety margin means effective expiry = now + 10s - 30s = now - 20s,
		// the cache is already stale on the second call.
		callCount := int64(0)
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			n := atomic.AddInt64(&callCount, 1)
			if n == 1 {
				return auth.Token{
					AccessToken: "token-1",
					ExpiresAt:   time.Now().Add(10 * time.Second),
				}, nil
			}
			return auth.Token{
				AccessToken: "token-2",
				ExpiresAt:   time.Now().Add(time.Hour),
			}, nil
		})

		tok1, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("call 1: unexpected error: %v", err)
		}
		if tok1.AccessToken != "token-1" {
			t.Errorf("call 1: AccessToken = %q; want %q", tok1.AccessToken, "token-1")
		}

		// Second call should see cache as stale (ExpiresAt - 30s is in the past).
		tok2, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("call 2: unexpected error: %v", err)
		}
		if tok2.AccessToken != "token-2" {
			t.Errorf("call 2: AccessToken = %q; want %q", tok2.AccessToken, "token-2")
		}
		if c := atomic.LoadInt64(&callCount); c != 2 {
			t.Errorf("fn called %d times; want 2 (cache miss on expiry)", c)
		}
	})

	t.Run("zero ExpiresAt means never expires (cache forever)", func(t *testing.T) {
		var calls int64
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			atomic.AddInt64(&calls, 1)
			return auth.Token{AccessToken: "forever", ExpiresAt: time.Time{}}, nil
		})

		for i := 0; i < 20; i++ {
			tok, err := cp.Token(context.Background())
			if err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
			if tok.AccessToken != "forever" {
				t.Errorf("call %d: AccessToken = %q; want %q", i, tok.AccessToken, "forever")
			}
		}
		if c := atomic.LoadInt64(&calls); c != 1 {
			t.Errorf("fn called %d times; want 1 (zero-expiry caches forever)", c)
		}
	})

	t.Run("error propagation without cache update", func(t *testing.T) {
		providerErr := errors.New("provider failure")
		callCount := int64(0)
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			n := atomic.AddInt64(&callCount, 1)
			if n == 1 {
				return auth.Token{}, providerErr
			}
			return auth.Token{AccessToken: "recovered", ExpiresAt: time.Now().Add(time.Hour)}, nil
		})

		// First call: error returned, cache not populated.
		_, err := cp.Token(context.Background())
		if !errors.Is(err, providerErr) {
			t.Fatalf("call 1: error = %v; want %v", err, providerErr)
		}

		// Second call: fn invoked again (cache was not updated).
		tok, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("call 2: unexpected error: %v", err)
		}
		if tok.AccessToken != "recovered" {
			t.Errorf("call 2: AccessToken = %q; want %q", tok.AccessToken, "recovered")
		}
		if c := atomic.LoadInt64(&callCount); c != 2 {
			t.Errorf("fn called %d times; want 2", c)
		}
	})

	t.Run("concurrent single-flight semantics", func(t *testing.T) {
		var calls int64
		// Use a channel to hold all goroutines until they are all ready.
		ready := make(chan struct{})
		cp := auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			atomic.AddInt64(&calls, 1)
			// Simulate a slow provider so concurrent callers pile up.
			time.Sleep(50 * time.Millisecond)
			return auth.Token{AccessToken: "single", ExpiresAt: time.Now().Add(time.Hour)}, nil
		})

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)
		errs := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				<-ready
				tok, err := cp.Token(context.Background())
				if err != nil {
					errs <- err
					return
				}
				if tok.AccessToken != "single" {
					errs <- errors.New("unexpected access token: " + tok.AccessToken)
				}
			}()
		}
		close(ready) // Release all goroutines simultaneously.
		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("goroutine error: %v", err)
		}
		if c := atomic.LoadInt64(&calls); c != 1 {
			t.Errorf("fn called %d times; want exactly 1 (single-flight)", c)
		}
	})

	t.Run("satisfies CredentialProvider interface", func(t *testing.T) {
		var _ auth.CredentialProvider = auth.RefreshFunc(func(_ context.Context) (auth.Token, error) {
			return auth.Token{}, nil
		})
	})

	t.Run("context is forwarded to fn", func(t *testing.T) {
		type ctxKey struct{}
		cp := auth.RefreshFunc(func(ctx context.Context) (auth.Token, error) {
			v, ok := ctx.Value(ctxKey{}).(string)
			if !ok || v != "hello" {
				return auth.Token{}, errors.New("context value missing or wrong")
			}
			return auth.Token{AccessToken: "ctx-ok"}, nil
		})

		ctx := context.WithValue(context.Background(), ctxKey{}, "hello")
		tok, err := cp.Token(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "ctx-ok" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "ctx-ok")
		}
	})
}
