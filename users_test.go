package intelligencecloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// ---------------------------------------------------------------------------
// [B-21] UserService.PatchCompany tests
// ---------------------------------------------------------------------------

func TestUserService_PatchCompany_Success(t *testing.T) {
	resp := generated.CompanyUserResponse{
		Data: generated.CompanyUserResponseData{
			Id:          strPtr("user-123"),
			Email:       strPtr("alice@example.com"),
			FirstName:   strPtr("Alice"),
			LastName:    strPtr("Smith"),
			DisplayName: strPtr("Alice Smith"),
			Role:        strPtr("admin"),
			Status:      strPtr("active"),
			Department:  strPtr("Engineering"),
			PhoneNumber: strPtr("+15551234567"),
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/companies/comp-1/users/user-123" {
			t.Errorf("path = %s, want /api/v1/companies/comp-1/users/user-123", r.URL.Path)
		}

		// Verify request body is valid JSON.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		var reqBody generated.UpdateCompanyUserRequest
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if reqBody.Department == nil || *reqBody.Department != "Engineering" {
			t.Errorf("department = %v, want 'Engineering'", reqBody.Department)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	dept := "Engineering"
	req := UpdateCompanyUserRequest{
		Department: &dept,
	}

	got, err := c.Users.PatchCompany(context.Background(), "comp-1", "user-123", req)
	if err != nil {
		t.Fatalf("PatchCompany: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}
	if got.Id == nil || *got.Id != "user-123" {
		t.Errorf("Id = %v, want 'user-123'", got.Id)
	}
	if got.Department == nil || *got.Department != "Engineering" {
		t.Errorf("Department = %v, want 'Engineering'", got.Department)
	}
	if got.Email == nil || *got.Email != "alice@example.com" {
		t.Errorf("Email = %v, want 'alice@example.com'", got.Email)
	}
}

func TestUserService_PatchCompany_400_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "validation_error",
			"message": "invalid phone number",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "comp-1", "user-1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

func TestUserService_PatchCompany_401_AuthenticationError(t *testing.T) {
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

	_, err = c.Users.PatchCompany(context.Background(), "comp-1", "user-1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("expected ErrAuthentication, got %v", err)
	}
}

func TestUserService_PatchCompany_403_AuthorizationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "forbidden",
			"message": "not authorized",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "comp-1", "user-1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrAuthorization) {
		t.Errorf("expected ErrAuthorization, got %v", err)
	}
}

func TestUserService_PatchCompany_404_NotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "not_found",
			"message": "user not found",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "comp-1", "nonexistent", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected *NotFoundError, got %T", err)
	}
}

func TestUserService_PatchCompany_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.CompanyUserResponse{})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = c.Users.PatchCompany(ctx, "comp-1", "user-1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestUserService_PatchCompany_EmptyCompanyID(t *testing.T) {
	c, err := NewClient("http://localhost:9999", auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "", "user-1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for empty companyID")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError for empty companyID, got %T: %v", err, err)
	}
}

func TestUserService_PatchCompany_EmptyUserID(t *testing.T) {
	c, err := NewClient("http://localhost:9999", auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "comp-1", "", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for empty userID")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError for empty userID, got %T: %v", err, err)
	}
}

func TestUserService_PatchCompany_CallOption_ExtraHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.CompanyUserResponse{
			Data: generated.CompanyUserResponseData{Id: strPtr("u1")},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "c1", "u1", UpdateCompanyUserRequest{},
		WithExtraHeader("X-Custom", "val"))
	if err != nil {
		t.Fatalf("PatchCompany: %v", err)
	}
	if gotHeader != "val" {
		t.Errorf("X-Custom = %q, want %q", gotHeader, "val")
	}
}

func TestUserService_PatchCompany_BearerTokenSent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.CompanyUserResponse{
			Data: generated.CompanyUserResponseData{Id: strPtr("u1")},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("my-token"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "c1", "u1", UpdateCompanyUserRequest{})
	if err != nil {
		t.Fatalf("PatchCompany: %v", err)
	}
	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-token")
	}
}

func TestUserService_PatchCompany_WithRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.CompanyUserResponse{
			Data: generated.CompanyUserResponseData{Id: strPtr("u1")},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.Users.PatchCompany(context.Background(), "c1", "u1", UpdateCompanyUserRequest{},
		WithRequestTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("PatchCompany: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}
}

func TestUserService_PatchCompany_500_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "internal",
			"message": "server error",
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "c1", "u1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !errors.Is(err, ErrServer) {
		t.Errorf("expected ErrServer, got %v", err)
	}
}

func TestUserService_PatchCompany_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, auth.StaticToken("tok"), WithDisableRetry())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Users.PatchCompany(context.Background(), "c1", "u1", UpdateCompanyUserRequest{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// strPtr is a test helper that returns a pointer to s.
func strPtr(s string) *string {
	return &s
}
