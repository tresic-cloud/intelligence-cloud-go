package auth_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

func TestStaticToken(t *testing.T) {
	t.Run("returns supplied token with zero expiry", func(t *testing.T) {
		cp := auth.StaticToken("my-bearer-token")
		tok, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "my-bearer-token" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "my-bearer-token")
		}
		if !tok.ExpiresAt.IsZero() {
			t.Errorf("ExpiresAt = %v; want zero time (never expires)", tok.ExpiresAt)
		}
	})

	t.Run("returns same token on repeated calls", func(t *testing.T) {
		cp := auth.StaticToken("repeat-token")
		for i := 0; i < 10; i++ {
			tok, err := cp.Token(context.Background())
			if err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
			if tok.AccessToken != "repeat-token" {
				t.Errorf("call %d: AccessToken = %q; want %q", i, tok.AccessToken, "repeat-token")
			}
			if !tok.ExpiresAt.Equal(time.Time{}) {
				t.Errorf("call %d: ExpiresAt = %v; want zero time", i, tok.ExpiresAt)
			}
		}
	})

	t.Run("accepts empty string without panic", func(t *testing.T) {
		cp := auth.StaticToken("")
		tok, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "" {
			t.Errorf("AccessToken = %q; want empty string", tok.AccessToken)
		}
	})

	t.Run("ignores cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cp := auth.StaticToken("ctx-token")
		tok, err := cp.Token(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "ctx-token" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "ctx-token")
		}
	})

	t.Run("satisfies CredentialProvider interface", func(t *testing.T) {
		var _ auth.CredentialProvider = auth.StaticToken("check")
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		cp := auth.StaticToken("concurrent-token")
		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)
		errs := make(chan error, goroutines)
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				tok, err := cp.Token(context.Background())
				if err != nil {
					errs <- err
					return
				}
				if tok.AccessToken != "concurrent-token" {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("concurrent call error: %v", err)
		}
	})
}
