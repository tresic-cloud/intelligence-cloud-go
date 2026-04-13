package intelligencecloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	genapi "github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// ---------------------------------------------------------------------------
// [B-13] AuthService tests — Login with OAuth error envelope
// ---------------------------------------------------------------------------

func TestAuthService_Login_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify NO Authorization header is sent.
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("expected /api/v1/auth/login, got %s", r.URL.Path)
		}

		// Decode request body.
		var body genapi.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Email != "user@example.com" {
			t.Errorf("email = %q, want %q", body.Email, "user@example.com")
		}
		if body.Password != "secret123" {
			t.Errorf("password = %q, want %q", body.Password, "secret123")
		}

		resp := genapi.LoginResponse{
			AccessToken:  "eyJ.access.token",
			RefreshToken: "eyJ.refresh.token",
			ExpiresIn:    3600,
			TokenType:    genapi.LoginResponseTokenTypeBearer,
			User: genapi.UserProfile{
				Id:          "user-uuid-1",
				Email:       "user@example.com",
				FirstName:   "Test",
				LastName:    "User",
				DisplayName: "Test User",
				Role:        "admin",
				Status:      genapi.UserProfileStatusActive,
				UserType:    genapi.UserProfileUserTypeResellerUser,
				Scopes:      []string{"read", "write"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	// Override httpClient to bypass transport (no auth injection).
	client.httpClient = ts.Client()

	resp, err := client.Auth.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	if resp.AccessToken != "eyJ.access.token" {
		t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "eyJ.access.token")
	}
	if resp.RefreshToken != "eyJ.refresh.token" {
		t.Errorf("RefreshToken = %q, want %q", resp.RefreshToken, "eyJ.refresh.token")
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, 3600)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", resp.TokenType, "Bearer")
	}
	if resp.User.Id != "user-uuid-1" {
		t.Errorf("User.Id = %q, want %q", resp.User.Id, "user-uuid-1")
	}
	if resp.User.Email != "user@example.com" {
		t.Errorf("User.Email = %q, want %q", resp.User.Email, "user@example.com")
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_InvalidCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify NO Authorization header.
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_credentials",
			"error_description": "The email or password is incorrect.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "bad@example.com",
		Password: "wrong",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should map to AuthenticationError.
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthenticationError, got %T: %v", err, err)
	}
	if authErr.Status() != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", authErr.Status(), http.StatusUnauthorized)
	}
	if authErr.Code() != "invalid_credentials" {
		t.Errorf("Code = %q, want %q", authErr.Code(), "invalid_credentials")
	}
	if authErr.Operation() != "Login" {
		t.Errorf("Operation = %q, want %q", authErr.Operation(), "Login")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Error("expected error to wrap ErrAuthentication")
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_UserNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "user_not_found",
			"error_description": "No account found with that email.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "missing@example.com",
		Password: "any",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Status() != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", nfErr.Status(), http.StatusNotFound)
	}
	if nfErr.Code() != "user_not_found" {
		t.Errorf("Code = %q, want %q", nfErr.Code(), "user_not_found")
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_AccountDeactivated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "account_deactivated",
			"error_description": "This account has been deactivated.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "deactivated@example.com",
		Password: "any",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Status() != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", valErr.Status(), http.StatusBadRequest)
	}
	if valErr.Code() != "account_deactivated" {
		t.Errorf("Code = %q, want %q", valErr.Code(), "account_deactivated")
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_ServiceUnavailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "authentication_service_unavailable",
			"error_description": "The authentication service is temporarily unavailable.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "any@example.com",
		Password: "any",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
	if srvErr.Status() != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", srvErr.Status(), http.StatusServiceUnavailable)
	}
	if srvErr.Code() != "authentication_service_unavailable" {
		t.Errorf("Code = %q, want %q", srvErr.Code(), "authentication_service_unavailable")
	}
}

func TestAuthService_Login_StandardErrorEnvelope(t *testing.T) {
	// Test that the standard ErrorResponse envelope is also handled.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
				Code:    "invalid_request",
				Message: "email is required",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "",
		Password: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Code() != "invalid_request" {
		t.Errorf("Code = %q, want %q", valErr.Code(), "invalid_request")
	}
}

func TestAuthService_Login_NoAuthorizationHeader(t *testing.T) {
	// Specifically verify the Authorization header is never sent.
	authHeaderSeen := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			authHeaderSeen = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(genapi.LoginResponse{
			AccessToken:  "tok",
			RefreshToken: "ref",
			ExpiresIn:    3600,
			TokenType:    genapi.LoginResponseTokenTypeBearer,
			User: genapi.UserProfile{
				Id:          "u1",
				Email:       "a@b.com",
				FirstName:   "A",
				LastName:    "B",
				DisplayName: "AB",
				Role:        "admin",
				Status:      genapi.UserProfileStatusActive,
				UserType:    genapi.UserProfileUserTypeTresicAdmin,
				Scopes:      []string{},
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("should-not-appear"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if authHeaderSeen {
		t.Error("Authorization header was sent, but Login should bypass auth")
	}
}

func TestAuthService_Login_InvalidResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_Forbidden(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "insufficient_scope",
			"error_description": "You do not have permission.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var authzErr *AuthorizationError
	if !errors.As(err, &authzErr) {
		t.Fatalf("expected *AuthorizationError, got %T: %v", err, err)
	}
	if authzErr.Status() != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", authzErr.Status(), http.StatusForbidden)
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_Conflict(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "conflict",
			"error_description": "Resource conflict",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_RateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "rate_limited",
			"error_description": "Too many requests.",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_UnexpectedStatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot) // 418
		w.Write([]byte(`{"error":"teapot","error_description":"I am a teapot"}`))
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var unexpErr *UnexpectedError
	if !errors.As(err, &unexpErr) {
		t.Fatalf("expected *UnexpectedError, got %T: %v", err, err)
	}
	if unexpErr.Status() != http.StatusTeapot {
		t.Errorf("Status = %d, want %d", unexpErr.Status(), http.StatusTeapot)
	}
}

func TestAuthService_Login_OAuthErrorEnvelope_Validation422(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_request",
			"error_description": "Unprocessable entity",
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Status() != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", valErr.Status(), http.StatusUnprocessableEntity)
	}
}

func TestAuthService_Login_EmptyErrorBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_WithCallOptions(t *testing.T) {
	headerSeen := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerSeen = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(genapi.LoginResponse{
			AccessToken:  "tok",
			RefreshToken: "ref",
			ExpiresIn:    3600,
			TokenType:    genapi.LoginResponseTokenTypeBearer,
			User: genapi.UserProfile{
				Id: "u1", Email: "a@b.com", FirstName: "A", LastName: "B",
				DisplayName: "AB", Role: "admin", Status: genapi.UserProfileStatusActive,
				UserType: genapi.UserProfileUserTypeTresicAdmin, Scopes: []string{},
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	_, err = client.Auth.Login(context.Background(), LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	}, WithExtraHeader("X-Custom", "test-value"))
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if headerSeen != "test-value" {
		t.Errorf("X-Custom = %q, want %q", headerSeen, "test-value")
	}
}

func TestAuthService_Login_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("ignored-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err = client.Auth.Login(ctx, LoginRequest{
		Email:    "a@b.com",
		Password: "p",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
