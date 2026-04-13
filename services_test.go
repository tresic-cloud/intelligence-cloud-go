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

// strPtr is a test helper that returns a pointer to s.
func strPtr(s string) *string {
	return &s
}

// ─── MeService ───────────────────────────────────────────────────

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

// ─── ResellerService ─────────────────────────────────────────────

// ---------------------------------------------------------------------------
// [B-11] ResellerService.List — paginated
// ---------------------------------------------------------------------------

func TestResellerService_List_SinglePage(t *testing.T) {
	t.Parallel()
	name1 := "Reseller A"
	name2 := "Reseller B"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers" {
			t.Errorf("path = %s, want /api/v1/resellers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data: []generated.Reseller{
				{ResellerName: &name1},
				{ResellerName: &name2},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	var names []string
	for it.Next(context.Background()) {
		r := it.Value()
		if r.ResellerName != nil {
			names = append(names, *r.ResellerName)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d items, want 2", len(names))
	}
	if names[0] != "Reseller A" || names[1] != "Reseller B" {
		t.Errorf("names = %v", names)
	}
}

func TestResellerService_List_MultiPage(t *testing.T) {
	t.Parallel()
	name1 := "Page1"
	name2 := "Page2"
	cursor := "cursor-2"
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			json.NewEncoder(w).Encode(generated.ResellerListResponse{
				Data:       []generated.Reseller{{ResellerName: &name1}},
				HasMore:    true,
				NextCursor: &cursor,
			})
		case 2:
			if r.URL.Query().Get("cursor") != "cursor-2" {
				t.Errorf("cursor = %q, want %q", r.URL.Query().Get("cursor"), "cursor-2")
			}
			json.NewEncoder(w).Encode(generated.ResellerListResponse{
				Data:    []generated.Reseller{{ResellerName: &name2}},
				HasMore: false,
			})
		default:
			t.Error("unexpected page request")
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	var names []string
	for it.Next(context.Background()) {
		r := it.Value()
		if r.ResellerName != nil {
			names = append(names, *r.ResellerName)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d items, want 2", len(names))
	}
	if names[0] != "Page1" || names[1] != "Page2" {
		t.Errorf("names = %v", names)
	}
}

func TestResellerService_List_WithPageSize(t *testing.T) {
	t.Parallel()
	name := "R1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limit := r.URL.Query().Get("limit"); limit != "5" {
			t.Errorf("limit = %q, want %q", limit, "5")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data:    []generated.Reseller{{ResellerName: &name}},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background(), WithPageSize(5))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_List_WithFilter(t *testing.T) {
	t.Parallel()
	name := "US Reseller"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if country := r.URL.Query().Get("country"); country != "US" {
			t.Errorf("country = %q, want %q", country, "US")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data:    []generated.Reseller{{ResellerName: &name}},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background(), WithFilter("country", "US"))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_List_ErrorMapping(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
		w.Write(errorJSON("FORBIDDEN", "insufficient scopes", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	if it.Next(context.Background()) {
		t.Error("expected Next to return false on error")
	}
	err := it.Err()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthorization) {
		t.Errorf("errors.Is(err, ErrAuthorization) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Get
// ---------------------------------------------------------------------------

func TestResellerService_Get_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Acme Corp"
	active := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
			IsActive:     &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Get(context.Background(), "reseller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.Id == nil || *reseller.Id != "reseller-1" {
		t.Errorf("Id = %v, want reseller-1", reseller.Id)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Acme Corp" {
		t.Errorf("ResellerName = %v, want Acme Corp", reseller.ResellerName)
	}
}

func TestResellerService_Get_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "nonexistent")
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

func TestResellerService_Get_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

func TestResellerService_Get_500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write(errorJSON("INTERNAL", "server error", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
	if !errors.Is(err, ErrServer) {
		t.Errorf("errors.Is(err, ErrServer) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Create
// ---------------------------------------------------------------------------

func TestResellerService_Create_Success(t *testing.T) {
	t.Parallel()
	id := "new-reseller"
	name := "New Reseller"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers" {
			t.Errorf("path = %s, want /api/v1/resellers", r.URL.Path)
		}

		var body generated.CreateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ResellerName != "New Reseller" {
			t.Errorf("ResellerName = %q, want %q", body.ResellerName, "New Reseller")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Create(context.Background(), CreateResellerRequest{
		ResellerName:        "New Reseller",
		PrimaryContactName:  "Jane Doe",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.Id == nil || *reseller.Id != "new-reseller" {
		t.Errorf("Id = %v, want new-reseller", reseller.Id)
	}
}

func TestResellerService_Create_409(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		w.Write(errorJSON("CONFLICT", "reseller name already exists", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{
		ResellerName:        "Duplicate",
		PrimaryContactName:  "Jane",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
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

func TestResellerService_Create_422(t *testing.T) {
	t.Parallel()
	field := "reseller_name"
	fieldCode := "required"
	fieldMsg := "reseller_name is required"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{})
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
	if ve.FieldErrors[0].Field != "reseller_name" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "reseller_name")
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Update
// ---------------------------------------------------------------------------

func TestResellerService_Update_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Updated Name"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}

		var body generated.CreateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ResellerName != "Updated Name" {
			t.Errorf("ResellerName = %q, want %q", body.ResellerName, "Updated Name")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Update(context.Background(), "reseller-1", UpdateResellerRequest{
		ResellerName:        "Updated Name",
		PrimaryContactName:  "Jane",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Updated Name" {
		t.Errorf("ResellerName = %v, want Updated Name", reseller.ResellerName)
	}
}

func TestResellerService_Update_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Update(context.Background(), "nonexistent", UpdateResellerRequest{
		ResellerName:        "X",
		PrimaryContactName:  "Y",
		PrimaryContactEmail: "y@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Patch
// ---------------------------------------------------------------------------

func TestResellerService_Patch_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Patched Name"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req PatchResellerRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.ResellerName == nil || *req.ResellerName != "Patched Name" {
			t.Errorf("ResellerName = %v, want Patched Name", req.ResellerName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	patchName := "Patched Name"
	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Patch(context.Background(), "reseller-1", PatchResellerRequest{
		ResellerName: &patchName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Patched Name" {
		t.Errorf("ResellerName = %v, want Patched Name", reseller.ResellerName)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Deactivate
// ---------------------------------------------------------------------------

func TestResellerService_Deactivate_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	active := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/deactivate" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/deactivate", r.URL.Path)
		}

		var body generated.DeactivateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Reason != "customer request" {
			t.Errorf("Reason = %q, want %q", body.Reason, "customer request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:       &id,
			IsActive: &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Deactivate(context.Background(), "reseller-1", DeactivateResellerRequest{
		Reason: "customer request",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.IsActive == nil || *reseller.IsActive != false {
		t.Errorf("IsActive = %v, want false", reseller.IsActive)
	}
}

func TestResellerService_Deactivate_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Deactivate(context.Background(), "nonexistent", DeactivateResellerRequest{
		Reason: "test",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

func TestResellerService_Deactivate_422(t *testing.T) {
	t.Parallel()
	field := "reason"
	fieldCode := "required"
	fieldMsg := "reason is required"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Deactivate(context.Background(), "reseller-1", DeactivateResellerRequest{})
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
	if ve.FieldErrors[0].Field != "reason" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "reason")
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Reactivate
// ---------------------------------------------------------------------------

func TestResellerService_Reactivate_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	active := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/reactivate" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/reactivate", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:       &id,
			IsActive: &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Reactivate(context.Background(), "reseller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.IsActive == nil || !*reseller.IsActive {
		t.Errorf("IsActive = %v, want true", reseller.IsActive)
	}
}

func TestResellerService_Reactivate_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Reactivate(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.ListAuditLogs
// ---------------------------------------------------------------------------

func TestResellerService_ListAuditLogs_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/audit-logs" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/audit-logs", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AuditLogListResponse{
			Data: []generated.AuditLogEntry{
				{
					Id:        "log-1",
					Action:    "reseller.updated",
					ActorId:   "user-1",
					ActorName: "Admin",
					Details:   "Updated reseller name",
					Timestamp: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1")
	defer it.Close()

	var entries []AuditLogEntry
	for it.Next(context.Background()) {
		entries = append(entries, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Action != "reseller.updated" {
		t.Errorf("Action = %q, want %q", entries[0].Action, "reseller.updated")
	}
}

func TestResellerService_ListAuditLogs_WithFilter(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if action := r.URL.Query().Get("action"); action != "reseller.deactivated" {
			t.Errorf("action = %q, want %q", action, "reseller.deactivated")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AuditLogListResponse{
			Data:    []generated.AuditLogEntry{},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1",
		WithFilter("action", "reseller.deactivated"))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_ListAuditLogs_ErrorMapping(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Error("expected Next to return false on error")
	}
	err := it.Err()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService — 429 Rate Limit
// ---------------------------------------------------------------------------

func TestResellerService_Get_429(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(429)
		w.Write(errorJSON("RATE_LIMITED", "too many requests", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
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
	if rl.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want 60s", rl.RetryAfter)
	}
}

// ---------------------------------------------------------------------------
// ResellerService — 400 Bad Request
// ---------------------------------------------------------------------------

func TestResellerService_Create_400(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		w.Write(errorJSON("BAD_REQUEST", "malformed request body", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
}

// ─── AuthService ─────────────────────────────────────────────────

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
		var body generated.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Email != "user@example.com" {
			t.Errorf("email = %q, want %q", body.Email, "user@example.com")
		}
		if body.Password != "secret123" {
			t.Errorf("password = %q, want %q", body.Password, "secret123")
		}

		resp := generated.LoginResponse{
			AccessToken:  "eyJ.access.token",
			RefreshToken: "eyJ.refresh.token",
			ExpiresIn:    3600,
			TokenType:    generated.LoginResponseTokenTypeBearer,
			User: generated.UserProfile{
				Id:          "user-uuid-1",
				Email:       "user@example.com",
				FirstName:   "Test",
				LastName:    "User",
				DisplayName: "Test User",
				Role:        "admin",
				Status:      generated.UserProfileStatusActive,
				UserType:    generated.UserProfileUserTypeResellerUser,
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
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
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
		json.NewEncoder(w).Encode(generated.LoginResponse{
			AccessToken:  "tok",
			RefreshToken: "ref",
			ExpiresIn:    3600,
			TokenType:    generated.LoginResponseTokenTypeBearer,
			User: generated.UserProfile{
				Id:          "u1",
				Email:       "a@b.com",
				FirstName:   "A",
				LastName:    "B",
				DisplayName: "AB",
				Role:        "admin",
				Status:      generated.UserProfileStatusActive,
				UserType:    generated.UserProfileUserTypeTresicAdmin,
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
		json.NewEncoder(w).Encode(generated.LoginResponse{
			AccessToken:  "tok",
			RefreshToken: "ref",
			ExpiresIn:    3600,
			TokenType:    generated.LoginResponseTokenTypeBearer,
			User: generated.UserProfile{
				Id: "u1", Email: "a@b.com", FirstName: "A", LastName: "B",
				DisplayName: "AB", Role: "admin", Status: generated.UserProfileStatusActive,
				UserType: generated.UserProfileUserTypeTresicAdmin, Scopes: []string{},
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

// ─── AuditLogService ─────────────────────────────────────────────

// ---------------------------------------------------------------------------
// [B-15] AuditLogService tests — paginated list with filters
// ---------------------------------------------------------------------------

func TestAuditLogService_ListForReseller_SinglePage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/audit-logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		resp := generated.AuditLogListResponse{
			Data: []generated.AuditLogEntry{
				{
					Id:        "entry-1",
					Action:    "user.created",
					ActorId:   "actor-1",
					ActorName: "Admin User",
					Details:   "Created user john@example.com",
					Timestamp: now,
				},
				{
					Id:        "entry-2",
					Action:    "user.updated",
					ActorId:   "actor-1",
					ActorName: "Admin User",
					Details:   "Updated user profile",
					Timestamp: now.Add(time.Hour),
				},
			},
			HasMore:    false,
			NextCursor: nil,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	var entries []AuditLogEntry
	for it.Next(context.Background()) {
		entries = append(entries, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Id != "entry-1" {
		t.Errorf("entries[0].Id = %q, want %q", entries[0].Id, "entry-1")
	}
	if entries[1].Action != "user.updated" {
		t.Errorf("entries[1].Action = %q, want %q", entries[1].Action, "user.updated")
	}
}

func TestAuditLogService_ListForReseller_Pagination(t *testing.T) {
	page := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")

		var resp generated.AuditLogListResponse
		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

		if page == 0 && cursor == "" {
			next := "cursor-page2"
			resp = generated.AuditLogListResponse{
				Data: []generated.AuditLogEntry{
					{Id: "e1", Action: "a1", ActorId: "u1", ActorName: "U1", Details: "d1", Timestamp: now},
				},
				HasMore:    true,
				NextCursor: &next,
			}
			page++
		} else if cursor == "cursor-page2" {
			resp = generated.AuditLogListResponse{
				Data: []generated.AuditLogEntry{
					{Id: "e2", Action: "a2", ActorId: "u2", ActorName: "U2", Details: "d2", Timestamp: now},
				},
				HasMore:    false,
				NextCursor: nil,
			}
		} else {
			t.Errorf("unexpected cursor: %q", cursor)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	var entries []AuditLogEntry
	for it.Next(context.Background()) {
		entries = append(entries, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Id != "e1" {
		t.Errorf("entries[0].Id = %q, want %q", entries[0].Id, "e1")
	}
	if entries[1].Id != "e2" {
		t.Errorf("entries[1].Id = %q, want %q", entries[1].Id, "e2")
	}

	pi := it.PageInfo()
	if pi.PagesFetched != 2 {
		t.Errorf("PagesFetched = %d, want 2", pi.PagesFetched)
	}
	if pi.HasNextPage {
		t.Error("HasNextPage should be false after exhausting all pages")
	}
}

func TestAuditLogService_ListForReseller_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Verify all filter params are passed correctly.
		if v := q.Get("action"); v != "user.created" {
			t.Errorf("action = %q, want %q", v, "user.created")
		}
		if v := q.Get("start_date"); v != "2026-01-01" {
			t.Errorf("start_date = %q, want %q", v, "2026-01-01")
		}
		if v := q.Get("end_date"); v != "2026-01-31" {
			t.Errorf("end_date = %q, want %q", v, "2026-01-31")
		}
		if v := q.Get("actor_id"); v != "actor-1" {
			t.Errorf("actor_id = %q, want %q", v, "actor-1")
		}
		if v := q.Get("search"); v != "john" {
			t.Errorf("search = %q, want %q", v, "john")
		}
		if v := q.Get("limit"); v != "10" {
			t.Errorf("limit = %q, want %q", v, "10")
		}

		resp := generated.AuditLogListResponse{
			Data:    []generated.AuditLogEntry{},
			HasMore: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1",
		WithFilter("action", "user.created"),
		WithFilter("start_date", "2026-01-01"),
		WithFilter("end_date", "2026-01-31"),
		WithFilter("actor_id", "actor-1"),
		WithFilter("search", "john"),
		WithPageSize(10),
	)
	defer it.Close()

	// Consume the iterator to trigger the request.
	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
}

func TestAuditLogService_ListForReseller_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-456")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "unauthorized",
				Message: "Invalid or expired token",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	err = it.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthenticationError, got %T: %v", err, err)
	}
	if authErr.Status() != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", authErr.Status(), http.StatusUnauthorized)
	}
}

func TestAuditLogService_ListForReseller_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generated.AuditLogListResponse{
			Data:    []generated.AuditLogEntry{},
			HasMore: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected false for empty result set")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditLogService_ListForReseller_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "internal_error",
				Message: "Something went wrong",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	err = it.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
	if srvErr.Status() != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", srvErr.Status(), http.StatusInternalServerError)
	}
}

func TestAuditLogService_ListForReseller_ForbiddenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "forbidden",
				Message: "Not authorized",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var authzErr *AuthorizationError
	if !errors.As(it.Err(), &authzErr) {
		t.Fatalf("expected *AuthorizationError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_NotFoundError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "not_found",
				Message: "Reseller not found",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "nonexistent")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var nfErr *NotFoundError
	if !errors.As(it.Err(), &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_ValidationError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "bad_request",
				Message: "Invalid date format",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1",
		WithFilter("start_date", "not-a-date"),
	)
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var valErr *ValidationError
	if !errors.As(it.Err(), &valErr) {
		t.Fatalf("expected *ValidationError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_RateLimitError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "rate_limited",
				Message: "Too many requests",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var rlErr *RateLimitError
	if !errors.As(it.Err(), &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_ConflictError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "conflict",
				Message: "Resource conflict",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var conflictErr *ConflictError
	if !errors.As(it.Err(), &conflictErr) {
		t.Fatalf("expected *ConflictError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_UnexpectedError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
		w.Write([]byte("i am a teapot"))
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	var unexpErr *UnexpectedError
	if !errors.As(it.Err(), &unexpErr) {
		t.Fatalf("expected *UnexpectedError, got %T", it.Err())
	}
}

func TestAuditLogService_ListForReseller_WithMaxItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		next := "cursor-2"
		resp := generated.AuditLogListResponse{
			Data: []generated.AuditLogEntry{
				{Id: "e1", Action: "a1", ActorId: "u1", ActorName: "U1", Details: "d1", Timestamp: now},
				{Id: "e2", Action: "a2", ActorId: "u2", ActorName: "U2", Details: "d2", Timestamp: now},
				{Id: "e3", Action: "a3", ActorId: "u3", ActorName: "U3", Details: "d3", Timestamp: now},
			},
			HasMore:    true,
			NextCursor: &next,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.AuditLogs.ListForReseller(context.Background(), "reseller-1",
		WithMaxItems(2),
	)
	defer it.Close()

	var count int
	for it.Next(context.Background()) {
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (capped by MaxItems)", count)
	}
}

// ─── ConnectorService ────────────────────────────────────────────

// ---------------------------------------------------------------------------
// [B-17] ConnectorService tests — location + company scopes
// ---------------------------------------------------------------------------

// --- ListByLocation ---

func TestConnectorService_ListByLocation_SinglePage(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/locations/loc-1/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := generated.LocationConnectorListResponse{
			Data: []generated.LocationConnectorItem{
				{
					Id:              "conn-1",
					Name:            "Google Reviews",
					Type:            generated.LocationConnectorItemTypeGoogleReviews,
					TypeDisplayName: "Google Reviews",
					Status:          generated.LocationConnectorItemStatusActive,
					StatusDisplay:   generated.LocationConnectorItemStatusDisplayConnected,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
			Pagination: generated.PaginationDTO{
				HasMore:    false,
				NextCursor: nil,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.Connectors.ListByLocation(context.Background(), "loc-1")
	defer it.Close()

	var connectors []LocationConnector
	for it.Next(context.Background()) {
		connectors = append(connectors, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	if connectors[0].Id != "conn-1" {
		t.Errorf("Id = %q, want %q", connectors[0].Id, "conn-1")
	}
	if connectors[0].Name != "Google Reviews" {
		t.Errorf("Name = %q, want %q", connectors[0].Name, "Google Reviews")
	}
}

func TestConnectorService_ListByLocation_Pagination(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	page := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")

		var resp generated.LocationConnectorListResponse
		if page == 0 && cursor == "" {
			next := "loc-page2"
			resp = generated.LocationConnectorListResponse{
				Data: []generated.LocationConnectorItem{
					{Id: "c1", Name: "C1", Type: generated.LocationConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.LocationConnectorItemStatusActive, StatusDisplay: generated.LocationConnectorItemStatusDisplayConnected, CreatedAt: now, UpdatedAt: now},
				},
				Pagination: generated.PaginationDTO{HasMore: true, NextCursor: &next},
			}
			page++
		} else if cursor == "loc-page2" {
			resp = generated.LocationConnectorListResponse{
				Data: []generated.LocationConnectorItem{
					{Id: "c2", Name: "C2", Type: generated.LocationConnectorItemTypeNetsapiens, TypeDisplayName: "NetSapiens", Status: generated.LocationConnectorItemStatusInactive, StatusDisplay: generated.LocationConnectorItemStatusDisplayInactive, CreatedAt: now, UpdatedAt: now},
				},
				Pagination: generated.PaginationDTO{HasMore: false, NextCursor: nil},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.Connectors.ListByLocation(context.Background(), "loc-1")
	defer it.Close()

	var connectors []LocationConnector
	for it.Next(context.Background()) {
		connectors = append(connectors, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(connectors))
	}
	if connectors[0].Id != "c1" {
		t.Errorf("connectors[0].Id = %q, want %q", connectors[0].Id, "c1")
	}
	if connectors[1].Id != "c2" {
		t.Errorf("connectors[1].Id = %q, want %q", connectors[1].Id, "c2")
	}
}

func TestConnectorService_ListByLocation_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if v := q.Get("search"); v != "ooma" {
			t.Errorf("search = %q, want %q", v, "ooma")
		}
		if v := q.Get("status"); v != "active" {
			t.Errorf("status = %q, want %q", v, "active")
		}
		if v := q.Get("limit"); v != "25" {
			t.Errorf("limit = %q, want %q", v, "25")
		}

		resp := generated.LocationConnectorListResponse{
			Data:       []generated.LocationConnectorItem{},
			Pagination: generated.PaginationDTO{HasMore: false},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.Connectors.ListByLocation(context.Background(), "loc-1",
		WithFilter("search", "ooma"),
		WithFilter("status", "active"),
		WithPageSize(25),
	)
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
}

func TestConnectorService_ListByLocation_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "forbidden",
				Message: "Insufficient permissions",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.Connectors.ListByLocation(context.Background(), "loc-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	err = it.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var authzErr *AuthorizationError
	if !errors.As(err, &authzErr) {
		t.Fatalf("expected *AuthorizationError, got %T: %v", err, err)
	}
}

// --- ListByCompany ---

func TestConnectorService_ListByCompany_SinglePage(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/companies/comp-1/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := generated.CompanyConnectorListResponse{
			Data: []generated.CompanyConnectorItem{
				{
					Id:              "conn-c1",
					Name:            "Ooma VoIP",
					Type:            generated.CompanyConnectorItemTypeOoma,
					TypeDisplayName: "Ooma",
					Status:          generated.CompanyConnectorItemStatusActive,
					StatusDisplay:   generated.CompanyConnectorItemStatusDisplayConnected,
					LocationId:      "loc-1",
					LocationName:    "Main Office",
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
			Errors: []generated.ConnectorErrorSummaryItem{
				{Name: "Broken Connector", StatusMessage: "API key expired"},
			},
			Pagination: generated.PaginationDTO{
				HasMore:    false,
				NextCursor: nil,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "comp-1")
	defer result.Iterator.Close()

	// Check error summary.
	if len(result.ErrorSummary) != 1 {
		t.Fatalf("expected 1 error summary, got %d", len(result.ErrorSummary))
	}
	if result.ErrorSummary[0].Name != "Broken Connector" {
		t.Errorf("ErrorSummary[0].Name = %q, want %q", result.ErrorSummary[0].Name, "Broken Connector")
	}
	if result.ErrorSummary[0].StatusMessage != "API key expired" {
		t.Errorf("ErrorSummary[0].StatusMessage = %q, want %q", result.ErrorSummary[0].StatusMessage, "API key expired")
	}

	// Check data iterator.
	var connectors []CompanyConnector
	for result.Iterator.Next(context.Background()) {
		connectors = append(connectors, result.Iterator.Value())
	}
	if err := result.Iterator.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	if connectors[0].Id != "conn-c1" {
		t.Errorf("Id = %q, want %q", connectors[0].Id, "conn-c1")
	}
	if connectors[0].LocationId != "loc-1" {
		t.Errorf("LocationId = %q, want %q", connectors[0].LocationId, "loc-1")
	}
}

func TestConnectorService_ListByCompany_NestedPagination(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	page := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")

		var resp generated.CompanyConnectorListResponse

		if page == 0 && cursor == "" {
			next := "comp-page2"
			resp = generated.CompanyConnectorListResponse{
				Data: []generated.CompanyConnectorItem{
					{Id: "cc1", Name: "CC1", Type: generated.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.CompanyConnectorItemStatusActive, StatusDisplay: generated.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				},
				Errors: []generated.ConnectorErrorSummaryItem{
					{Name: "Bad Connector", StatusMessage: "connection timeout"},
				},
				Pagination: generated.PaginationDTO{HasMore: true, NextCursor: &next},
			}
			page++
		} else if cursor == "comp-page2" {
			resp = generated.CompanyConnectorListResponse{
				Data: []generated.CompanyConnectorItem{
					{Id: "cc2", Name: "CC2", Type: generated.CompanyConnectorItemTypeGoogleReviews, TypeDisplayName: "Google Reviews", Status: generated.CompanyConnectorItemStatusError, StatusDisplay: generated.CompanyConnectorItemStatusDisplayError, LocationId: "l2", LocationName: "L2", CreatedAt: now, UpdatedAt: now},
				},
				Errors: []generated.ConnectorErrorSummaryItem{
					{Name: "Bad Connector", StatusMessage: "connection timeout"},
				},
				Pagination: generated.PaginationDTO{HasMore: false, NextCursor: nil},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "comp-1")
	defer result.Iterator.Close()

	// ErrorSummary should be set from the first page response.
	if len(result.ErrorSummary) != 1 {
		t.Fatalf("expected 1 error summary, got %d", len(result.ErrorSummary))
	}

	var connectors []CompanyConnector
	for result.Iterator.Next(context.Background()) {
		connectors = append(connectors, result.Iterator.Value())
	}
	if err := result.Iterator.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(connectors))
	}
	if connectors[0].Id != "cc1" {
		t.Errorf("connectors[0].Id = %q, want %q", connectors[0].Id, "cc1")
	}
	if connectors[1].Id != "cc2" {
		t.Errorf("connectors[1].Id = %q, want %q", connectors[1].Id, "cc2")
	}
}

func TestConnectorService_ListByCompany_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if v := q.Get("search"); v != "google" {
			t.Errorf("search = %q, want %q", v, "google")
		}
		if v := q.Get("status"); v != "error" {
			t.Errorf("status = %q, want %q", v, "error")
		}
		if v := q.Get("type"); v != "google_reviews" {
			t.Errorf("type = %q, want %q", v, "google_reviews")
		}
		if v := q.Get("limit"); v != "50" {
			t.Errorf("limit = %q, want %q", v, "50")
		}

		resp := generated.CompanyConnectorListResponse{
			Data:       []generated.CompanyConnectorItem{},
			Errors:     []generated.ConnectorErrorSummaryItem{},
			Pagination: generated.PaginationDTO{HasMore: false},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "comp-1",
		WithFilter("search", "google"),
		WithFilter("status", "error"),
		WithFilter("type", "google_reviews"),
		WithPageSize(50),
	)
	defer result.Iterator.Close()

	for result.Iterator.Next(context.Background()) {
	}
	if err := result.Iterator.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
}

func TestConnectorService_ListByCompany_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(generated.ErrorResponse{
			Error: generated.Error{
				Code:    "not_found",
				Message: "Company not found",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "nonexistent")

	// ErrorSummary is empty on error.
	if len(result.ErrorSummary) != 0 {
		t.Errorf("expected empty ErrorSummary, got %d items", len(result.ErrorSummary))
	}

	// Iterator should yield error.
	if result.Iterator.Next(context.Background()) {
		t.Fatal("expected Next to return false on error")
	}

	err = result.Iterator.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
}

func TestConnectorService_ListByCompany_EmptyErrorSummary(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generated.CompanyConnectorListResponse{
			Data: []generated.CompanyConnectorItem{
				{Id: "cc1", Name: "CC1", Type: generated.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.CompanyConnectorItemStatusActive, StatusDisplay: generated.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
			},
			Errors:     []generated.ConnectorErrorSummaryItem{},
			Pagination: generated.PaginationDTO{HasMore: false},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "comp-1")
	defer result.Iterator.Close()

	if len(result.ErrorSummary) != 0 {
		t.Errorf("expected empty ErrorSummary, got %d", len(result.ErrorSummary))
	}

	var connectors []CompanyConnector
	for result.Iterator.Next(context.Background()) {
		connectors = append(connectors, result.Iterator.Value())
	}
	if err := result.Iterator.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
}

func TestConnectorService_ListByLocation_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generated.LocationConnectorListResponse{
			Data:       []generated.LocationConnectorItem{},
			Pagination: generated.PaginationDTO{HasMore: false},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	it := client.Connectors.ListByLocation(context.Background(), "loc-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Fatal("expected false for empty result set")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectorService_ListByCompany_WithMaxItems(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next := "comp-page2"
		resp := generated.CompanyConnectorListResponse{
			Data: []generated.CompanyConnectorItem{
				{Id: "cc1", Name: "CC1", Type: generated.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.CompanyConnectorItemStatusActive, StatusDisplay: generated.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				{Id: "cc2", Name: "CC2", Type: generated.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.CompanyConnectorItemStatusActive, StatusDisplay: generated.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				{Id: "cc3", Name: "CC3", Type: generated.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: generated.CompanyConnectorItemStatusActive, StatusDisplay: generated.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
			},
			Errors:     []generated.ConnectorErrorSummaryItem{},
			Pagination: generated.PaginationDTO{HasMore: true, NextCursor: &next},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, auth.StaticToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.httpClient = ts.Client()

	result := client.Connectors.ListByCompany(context.Background(), "comp-1",
		WithMaxItems(2),
	)
	defer result.Iterator.Close()

	var count int
	for result.Iterator.Next(context.Background()) {
		count++
	}
	if err := result.Iterator.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (capped by MaxItems)", count)
	}
}

// ─── ProductService ──────────────────────────────────────────────

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

// ─── UserService ─────────────────────────────────────────────────

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
