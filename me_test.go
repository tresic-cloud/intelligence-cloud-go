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
// Helpers
// ---------------------------------------------------------------------------

// newTestClient constructs a Client pointing at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(srv.URL, auth.StaticToken("test-token"),
		WithDisableRetry(),
		WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// errorJSON returns a JSON-encoded ErrorResponse body.
func errorJSON(code, message string, details []generated.ErrorDetail) []byte {
	type errBody struct {
		Error struct {
			Code    string                  `json:"code"`
			Message string                  `json:"message"`
			Details []generated.ErrorDetail `json:"details,omitempty"`
		} `json:"error"`
	}
	b := errBody{}
	b.Error.Code = code
	b.Error.Message = message
	b.Error.Details = details
	data, _ := json.Marshal(b)
	return data
}

// ---------------------------------------------------------------------------
// [B-9] MeService.Get
// ---------------------------------------------------------------------------

func TestMeService_Get_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/me" {
			t.Errorf("path = %s, want /api/v1/me", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.UserProfileResponse{
			Data: generated.UserProfile{
				Id:          "user-1",
				Email:       "admin@test.com",
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Role:        "super_admin",
				Status:      generated.UserProfileStatusActive,
				UserType:    generated.UserProfileUserTypeTresicAdmin,
				Scopes:      []string{"admin:*"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	user, err := c.Me.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Id != "user-1" {
		t.Errorf("Id = %q, want %q", user.Id, "user-1")
	}
	if user.Email != "admin@test.com" {
		t.Errorf("Email = %q, want %q", user.Email, "admin@test.com")
	}
	if user.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q, want %q", user.DisplayName, "John Doe")
	}
}

func TestMeService_Get_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
	var ae *AuthenticationError
	if !errors.As(err, &ae) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if ae.Status() != 401 {
		t.Errorf("Status() = %d, want 401", ae.Status())
	}
}

func TestMeService_Get_403(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
		w.Write(errorJSON("FORBIDDEN", "insufficient scopes", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthorization) {
		t.Errorf("errors.Is(err, ErrAuthorization) = false; err = %v", err)
	}
	var ae *AuthorizationError
	if !errors.As(err, &ae) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
}

func TestMeService_Get_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "user not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
	var ne *NotFoundError
	if !errors.As(err, &ne) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
}

func TestMeService_Get_500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write(errorJSON("INTERNAL", "internal server error", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrServer) {
		t.Errorf("errors.Is(err, ErrServer) = false; err = %v", err)
	}
	var se *ServerError
	if !errors.As(err, &se) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if se.Status() != 500 {
		t.Errorf("Status() = %d, want 500", se.Status())
	}
}

func TestMeService_Get_429(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		w.Write(errorJSON("RATE_LIMITED", "too many requests", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRateLimit) {
		t.Errorf("errors.Is(err, ErrRateLimit) = false; err = %v", err)
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if rl.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v, want 30s", rl.RetryAfter)
	}
}

func TestMeService_Get_ContextCancellation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Should not be reached.
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.Me.Get(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// ---------------------------------------------------------------------------
// MeService.CreateAvatarUploadURL
// ---------------------------------------------------------------------------

func TestMeService_CreateAvatarUploadURL_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/me/avatar/upload-url" {
			t.Errorf("path = %s, want /api/v1/me/avatar/upload-url", r.URL.Path)
		}

		var body generated.AvatarUploadURLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ContentType != generated.AvatarUploadURLRequestContentTypeImagePng {
			t.Errorf("ContentType = %q, want %q", body.ContentType, generated.AvatarUploadURLRequestContentTypeImagePng)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AvatarUploadURLResponse{
			Data: generated.AvatarUploadURLResponseData{
				UploadUrl: "https://blob.example.com/upload?sas=token",
				AssetUrl:  "https://blob.example.com/avatars/abc.png",
				ExpiresAt: time.Now().Add(15 * time.Minute),
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	result, err := c.Me.CreateAvatarUploadURL(context.Background(), AvatarUploadURLRequest{
		ContentType: generated.AvatarUploadURLRequestContentTypeImagePng,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UploadUrl == "" {
		t.Error("UploadUrl is empty")
	}
	if result.AssetUrl == "" {
		t.Error("AssetUrl is empty")
	}
}

func TestMeService_CreateAvatarUploadURL_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.CreateAvatarUploadURL(context.Background(), AvatarUploadURLRequest{
		ContentType: generated.AvatarUploadURLRequestContentTypeImagePng,
	})
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// MeService.ConfirmAvatar
// ---------------------------------------------------------------------------

func TestMeService_ConfirmAvatar_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/me/avatar" {
			t.Errorf("path = %s, want /api/v1/me/avatar", r.URL.Path)
		}

		var body generated.AvatarConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.AssetUrl != "https://blob.example.com/avatars/abc.png" {
			t.Errorf("AssetUrl = %q, want expected value", body.AssetUrl)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AvatarResponse{
			Data: generated.AvatarResponseData{
				AvatarUrl: "https://blob.example.com/avatars/abc.png",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	result, err := c.Me.ConfirmAvatar(context.Background(), AvatarConfirmRequest{
		AssetUrl: "https://blob.example.com/avatars/abc.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AvatarUrl != "https://blob.example.com/avatars/abc.png" {
		t.Errorf("AvatarUrl = %q, want expected value", result.AvatarUrl)
	}
}

func TestMeService_ConfirmAvatar_422(t *testing.T) {
	t.Parallel()
	field := "asset_url"
	fieldCode := "invalid"
	fieldMsg := "asset_url is not a valid URL"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.ConfirmAvatar(context.Background(), AvatarConfirmRequest{
		AssetUrl: "bad-url",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if len(ve.FieldErrors) != 1 {
		t.Fatalf("FieldErrors len = %d, want 1", len(ve.FieldErrors))
	}
	if ve.FieldErrors[0].Field != "asset_url" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "asset_url")
	}
	if ve.FieldErrors[0].Code != "invalid" {
		t.Errorf("FieldErrors[0].Code = %q, want %q", ve.FieldErrors[0].Code, "invalid")
	}
}

// ---------------------------------------------------------------------------
// MeService.DeleteAvatar
// ---------------------------------------------------------------------------

func TestMeService_DeleteAvatar_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/me/avatar" {
			t.Errorf("path = %s, want /api/v1/me/avatar", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Me.DeleteAvatar(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMeService_DeleteAvatar_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Me.DeleteAvatar(context.Background())
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// MeService.GetNotificationPreferences
// ---------------------------------------------------------------------------

func TestMeService_GetNotificationPreferences_Success(t *testing.T) {
	t.Parallel()
	boolTrue := true
	boolFalse := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/me/notification-preferences" {
			t.Errorf("path = %s, want /api/v1/me/notification-preferences", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.NotificationPreferencesResponse{
			Data: generated.NotificationPreferences{
				EmailNotifications: &boolTrue,
				PushNotifications:  &boolFalse,
				DailyRecapEmail:    &boolTrue,
				MissedCallAlerts:   &boolTrue,
				TaskReminders:      &boolFalse,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	prefs, err := c.Me.GetNotificationPreferences(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefs.EmailNotifications == nil || !*prefs.EmailNotifications {
		t.Error("EmailNotifications should be true")
	}
	if prefs.PushNotifications == nil || *prefs.PushNotifications {
		t.Error("PushNotifications should be false")
	}
}

func TestMeService_GetNotificationPreferences_500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write(errorJSON("INTERNAL", "server error", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.GetNotificationPreferences(context.Background())
	if !errors.Is(err, ErrServer) {
		t.Errorf("errors.Is(err, ErrServer) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// MeService.PatchNotificationPreferences
// ---------------------------------------------------------------------------

func TestMeService_PatchNotificationPreferences_Success(t *testing.T) {
	t.Parallel()
	boolTrue := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/me/notification-preferences" {
			t.Errorf("path = %s, want /api/v1/me/notification-preferences", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req NotificationPreferencesPatch
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.EmailNotifications == nil || !*req.EmailNotifications {
			t.Error("EmailNotifications should be true in request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.NotificationPreferencesResponse{
			Data: generated.NotificationPreferences{
				EmailNotifications: &boolTrue,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	prefs, err := c.Me.PatchNotificationPreferences(context.Background(), NotificationPreferencesPatch{
		EmailNotifications: &boolTrue,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefs.EmailNotifications == nil || !*prefs.EmailNotifications {
		t.Error("EmailNotifications should be true")
	}
}

// ---------------------------------------------------------------------------
// MeService.GetDailyRecapPreferences
// ---------------------------------------------------------------------------

func TestMeService_GetDailyRecapPreferences_Success(t *testing.T) {
	t.Parallel()
	tz := "America/New_York"
	sendTime := "08:00"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/me/daily-recap-preferences" {
			t.Errorf("path = %s, want /api/v1/me/daily-recap-preferences", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.DailyRecapPreferencesResponse{
			Data: generated.DailyRecapPreferences{
				Timezone: &tz,
				SendTime: &sendTime,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	prefs, err := c.Me.GetDailyRecapPreferences(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefs.Timezone == nil || *prefs.Timezone != "America/New_York" {
		t.Errorf("Timezone = %v, want America/New_York", prefs.Timezone)
	}
	if prefs.SendTime == nil || *prefs.SendTime != "08:00" {
		t.Errorf("SendTime = %v, want 08:00", prefs.SendTime)
	}
}

func TestMeService_GetDailyRecapPreferences_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "expired token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.GetDailyRecapPreferences(context.Background())
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// MeService.PatchDailyRecapPreferences
// ---------------------------------------------------------------------------

func TestMeService_PatchDailyRecapPreferences_Success(t *testing.T) {
	t.Parallel()
	tz := "Europe/London"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/me/daily-recap-preferences" {
			t.Errorf("path = %s, want /api/v1/me/daily-recap-preferences", r.URL.Path)
		}

		var req DailyRecapPreferencesPatch
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Timezone == nil || *req.Timezone != "Europe/London" {
			t.Errorf("Timezone = %v, want Europe/London", req.Timezone)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.DailyRecapPreferencesResponse{
			Data: generated.DailyRecapPreferences{
				Timezone: &tz,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	prefs, err := c.Me.PatchDailyRecapPreferences(context.Background(), DailyRecapPreferencesPatch{
		Timezone: &tz,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefs.Timezone == nil || *prefs.Timezone != "Europe/London" {
		t.Errorf("Timezone = %v, want Europe/London", prefs.Timezone)
	}
}

func TestMeService_PatchDailyRecapPreferences_422(t *testing.T) {
	t.Parallel()
	field := "timezone"
	fieldCode := "invalid_timezone"
	fieldMsg := "timezone must be a valid IANA identifier"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	badTZ := "Invalid/Zone"
	c := newTestClient(t, srv)
	_, err := c.Me.PatchDailyRecapPreferences(context.Background(), DailyRecapPreferencesPatch{
		Timezone: &badTZ,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if len(ve.FieldErrors) != 1 {
		t.Fatalf("FieldErrors len = %d, want 1", len(ve.FieldErrors))
	}
	if ve.FieldErrors[0].Field != "timezone" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "timezone")
	}
}

// ---------------------------------------------------------------------------
// MeService — 409 Conflict
// ---------------------------------------------------------------------------

func TestMeService_ConfirmAvatar_409(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		w.Write(errorJSON("CONFLICT", "avatar already set", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Me.ConfirmAvatar(context.Background(), AvatarConfirmRequest{
		AssetUrl: "https://blob.example.com/avatars/abc.png",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("errors.Is(err, ErrConflict) = false; err = %v", err)
	}
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
}
