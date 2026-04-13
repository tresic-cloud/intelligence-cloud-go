package transport

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// --- C-1: redaction helper tests ---

func TestRedactURL_SensitiveQueryParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		rawURL   string
		excludes string
	}{
		{"token value stripped", "https://api.example.com/v1?token=abc123&page=1", "abc123"},
		{"secret value stripped", "https://api.example.com/v1?client_secret=xyz", "xyz"},
		{"key value stripped", "https://api.example.com/v1?api_key=deadbeef", "deadbeef"},
		{"password value stripped", "https://api.example.com/v1?password=hunter2", "hunter2"},
		{"case insensitive", "https://api.example.com/v1?TOKEN=sec&ok=y", "sec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}
			got := RedactURL(u)
			if strings.Contains(got, tt.excludes) {
				t.Errorf("RedactURL(%q) = %q, should not contain %q", tt.rawURL, got, tt.excludes)
			}
		})
	}
}

func TestRedactURL_PreservesNonSensitive(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("https://api.example.com/v1?page=1&limit=50")
	got := RedactURL(u)
	if !strings.Contains(got, "page=1") {
		t.Errorf("expected 'page=1' in %q", got)
	}
	if !strings.Contains(got, "limit=50") {
		t.Errorf("expected 'limit=50' in %q", got)
	}
}

func TestRedactURL_Nil(t *testing.T) {
	t.Parallel()
	if got := RedactURL(nil); got != "" {
		t.Errorf("RedactURL(nil) = %q, want empty", got)
	}
}

func TestRedactHeaders_BlocksSensitive(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Authorization", "Bearer secret123")
	h.Set("Cookie", "session=abc")
	h.Set("X-Api-Key", "key456")
	h.Set("X-Token", "tok789")
	h.Set("Content-Type", "application/json")
	h.Set("X-Request-Id", "req-1")

	got := RedactHeaders(h)

	// Sensitive headers must be absent.
	for _, name := range []string{"Authorization", "Cookie", "X-Api-Key", "X-Token"} {
		if _, ok := got[name]; ok {
			t.Errorf("header %q should have been redacted", name)
		}
	}

	// Safe headers must be present.
	if got["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got["Content-Type"], "application/json")
	}
	if got["X-Request-Id"] != "req-1" {
		t.Errorf("X-Request-Id = %q, want %q", got["X-Request-Id"], "req-1")
	}
}

func TestRedactHeaders_Empty(t *testing.T) {
	t.Parallel()
	got := RedactHeaders(http.Header{})
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestRedactedHeaders_DefaultContents(t *testing.T) {
	t.Parallel()
	expected := map[string]bool{
		"Authorization": true,
		"Cookie":        true,
		"X-Api-Key":     true,
		"X-Token":       true,
	}
	for _, h := range RedactedHeaders {
		if !expected[h] {
			t.Errorf("unexpected header %q in RedactedHeaders", h)
		}
		delete(expected, h)
	}
	for h := range expected {
		t.Errorf("missing header %q in RedactedHeaders", h)
	}
}

func TestRedactURL_CombinedSensitiveAndSafe(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("https://api.example.com/path?token=secret456&normal=ok&password=pw123")
	got := RedactURL(u)
	if strings.Contains(got, "secret456") {
		t.Error("token value should be redacted")
	}
	if strings.Contains(got, "pw123") {
		t.Error("password value should be redacted")
	}
	if !strings.Contains(got, "normal=ok") {
		t.Error("normal param should be preserved")
	}
}

func TestRedactHeaders_CaseInsensitiveBlocking(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	// Go's http.Header canonicalizes keys, so "authorization" becomes "Authorization".
	h.Set("authorization", "Bearer secret")
	h.Set("x-custom", "safe")

	got := RedactHeaders(h)
	for k := range got {
		if strings.EqualFold(k, "Authorization") {
			t.Error("authorization should be blocked")
		}
	}
	// x-custom should be present (canonicalized to X-Custom by Go).
	if _, ok := got["X-Custom"]; !ok {
		t.Error("X-Custom should be present")
	}
}
