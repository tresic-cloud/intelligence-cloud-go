package intelligencecloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// ---------------------------------------------------------------------------
// [B-19] ProductService.List tests
// ---------------------------------------------------------------------------

func TestProductService_List_Success(t *testing.T) {
	products := generated.ProductListResponse{
		Data: []generated.Product{
			{Key: "voice-ai", Name: "Voice AI"},
			{Key: "review-management", Name: "Review Management"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/products" {
			t.Errorf("path = %s, want /api/v1/products", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.Products.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Key != "voice-ai" {
		t.Errorf("got[0].Key = %q, want %q", got[0].Key, "voice-ai")
	}
	if got[0].Name != "Voice AI" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Voice AI")
	}
	if got[1].Key != "review-management" {
		t.Errorf("got[1].Key = %q, want %q", got[1].Key, "review-management")
	}
}

func TestProductService_List_EmptyList(t *testing.T) {
	products := generated.ProductListResponse{
		Data: []generated.Product{},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.Products.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestProductService_List_401_AuthenticationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "invalid_token",
			"message": "token expired",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("expected ErrAuthentication, got %v", err)
	}
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T", err)
	}
}

func TestProductService_List_403_AuthorizationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "forbidden",
			"message": "insufficient permissions",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrAuthorization) {
		t.Errorf("expected ErrAuthorization, got %v", err)
	}
	var azErr *AuthorizationError
	if !errors.As(err, &azErr) {
		t.Errorf("expected *AuthorizationError, got %T", err)
	}
}

func TestProductService_List_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ProductListResponse{Data: []generated.Product{}})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = c.Products.List(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestProductService_List_CallOption_ExtraHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom-Header")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ProductListResponse{
			Data: []generated.Product{{Key: "k", Name: "n"}},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background(), WithExtraHeader("X-Custom-Header", "test-value"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotHeader != "test-value" {
		t.Errorf("X-Custom-Header = %q, want %q", gotHeader, "test-value")
	}
}

func TestProductService_List_NilData_ReturnsEmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Response with null data field.
		w.Write([]byte(`{"data":null}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.Products.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil slice (empty), got nil")
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestProductService_List_BearerTokenSent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ProductListResponse{
			Data: []generated.Product{},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("my-secret-token"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-secret-token")
	}
}

func TestProductService_List_WithRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ProductListResponse{
			Data: []generated.Product{{Key: "k", Name: "n"}},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.Products.List(context.Background(), WithRequestTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len(got) = %d, want 1", len(got))
	}
}

func TestProductService_List_WithIdempotencyKey(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ProductListResponse{
			Data: []generated.Product{},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background(), WithIdempotencyKey("idem-123"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotKey != "idem-123" {
		t.Errorf("X-Idempotency-Key = %q, want %q", gotKey, "idem-123")
	}
}

func TestProductService_List_500_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "internal",
			"message": "something went wrong",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !errors.Is(err, ErrServer) {
		t.Errorf("expected ErrServer, got %v", err)
	}
}

func TestProductService_List_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Products.List(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
