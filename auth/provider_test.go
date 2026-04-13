package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

func TestTokenFields(t *testing.T) {
	t.Run("zero value has empty access token and zero expiry", func(t *testing.T) {
		var tok auth.Token
		if tok.AccessToken != "" {
			t.Errorf("zero-value AccessToken = %q; want empty string", tok.AccessToken)
		}
		if !tok.ExpiresAt.IsZero() {
			t.Errorf("zero-value ExpiresAt = %v; want zero time", tok.ExpiresAt)
		}
	})

	t.Run("populated token retains values", func(t *testing.T) {
		exp := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		tok := auth.Token{
			AccessToken: "abc123",
			ExpiresAt:   exp,
		}
		if tok.AccessToken != "abc123" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "abc123")
		}
		if !tok.ExpiresAt.Equal(exp) {
			t.Errorf("ExpiresAt = %v; want %v", tok.ExpiresAt, exp)
		}
	})

	t.Run("zero ExpiresAt means never expires", func(t *testing.T) {
		tok := auth.Token{AccessToken: "forever-token"}
		if !tok.ExpiresAt.IsZero() {
			t.Errorf("ExpiresAt should be zero (never expires); got %v", tok.ExpiresAt)
		}
	})
}

// mockProvider is a test double that satisfies CredentialProvider.
type mockProvider struct {
	token auth.Token
	err   error
}

func (m *mockProvider) Token(_ context.Context) (auth.Token, error) {
	return m.token, m.err
}

func TestCredentialProviderInterface(t *testing.T) {
	t.Run("mock satisfying interface returns token", func(t *testing.T) {
		exp := time.Now().Add(time.Hour)
		mp := &mockProvider{
			token: auth.Token{AccessToken: "test-token", ExpiresAt: exp},
		}
		var cp auth.CredentialProvider = mp
		tok, err := cp.Token(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "test-token" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "test-token")
		}
		if !tok.ExpiresAt.Equal(exp) {
			t.Errorf("ExpiresAt = %v; want %v", tok.ExpiresAt, exp)
		}
	})

	t.Run("mock satisfying interface returns error", func(t *testing.T) {
		wantErr := errors.New("auth failure")
		mp := &mockProvider{err: wantErr}
		var cp auth.CredentialProvider = mp
		_, err := cp.Token(context.Background())
		if !errors.Is(err, wantErr) {
			t.Errorf("error = %v; want %v", err, wantErr)
		}
	})

	t.Run("interface works with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		mp := &mockProvider{
			token: auth.Token{AccessToken: "ctx-token"},
		}
		var cp auth.CredentialProvider = mp
		tok, err := cp.Token(ctx)
		// The mock ignores ctx, so it should still return the token.
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.AccessToken != "ctx-token" {
			t.Errorf("AccessToken = %q; want %q", tok.AccessToken, "ctx-token")
		}
	})
}
