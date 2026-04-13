package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers: stub API server for me endpoints
// ---------------------------------------------------------------------------

func newMeStubServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// GET /api/v1/me — get profile
	mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":           "usr_001",
				"email":        "user@example.com",
				"first_name":   "Test",
				"last_name":    "User",
				"display_name": "Test User",
				"role":         "admin",
				"user_type":    "tresic_admin",
				"status":       "active",
				"scopes":       []string{"read", "write"},
			},
		})
	})

	// GET /api/v1/me/notification-preferences — get notification prefs
	mux.HandleFunc("GET /api/v1/me/notification-preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"email_notifications": true,
				"push_notifications":  false,
				"daily_recap_email":   true,
				"missed_call_alerts":  true,
				"task_reminders":      false,
			},
		})
	})

	// PATCH /api/v1/me/notification-preferences — update notification prefs
	mux.HandleFunc("PATCH /api/v1/me/notification-preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"email_notifications": true,
				"push_notifications":  true,
				"daily_recap_email":   true,
				"missed_call_alerts":  true,
				"task_reminders":      false,
			},
		})
	})

	// GET /api/v1/me/daily-recap-preferences — get daily recap prefs
	mux.HandleFunc("GET /api/v1/me/daily-recap-preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"timezone":    "America/New_York",
				"send_time":   "09:00",
				"date_format": "mm/dd/yyyy",
				"time_format": "12h",
			},
		})
	})

	// PATCH /api/v1/me/daily-recap-preferences — update daily recap prefs
	mux.HandleFunc("PATCH /api/v1/me/daily-recap-preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"timezone":    "UTC",
				"send_time":   "08:00",
				"date_format": "yyyy-mm-dd",
				"time_format": "24h",
			},
		})
	})

	// POST /api/v1/me/avatar/upload-url — create avatar upload URL
	mux.HandleFunc("POST /api/v1/me/avatar/upload-url", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"upload_url": "https://storage.example.com/upload?token=abc",
				"asset_url":  "https://cdn.example.com/avatars/usr_001.jpg",
				"expires_at": "2026-04-13T21:00:00Z",
			},
		})
	})

	// PATCH /api/v1/me/avatar — confirm avatar (PATCH to /api/v1/me/avatar per generated client)
	mux.HandleFunc("PATCH /api/v1/me/avatar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"avatar_url": "https://cdn.example.com/avatars/usr_001.jpg",
			},
		})
	})

	// DELETE /api/v1/me/avatar — delete avatar
	mux.HandleFunc("DELETE /api/v1/me/avatar", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// runMeCmd executes a me command against a stub server.
func runMeCmd(t *testing.T, serverURL string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {
				Name:        "default",
				Environment: "dev",
				BaseURL:     serverURL,
			},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{
			"default": "tok_test_me",
		},
	}

	rootCmd := NewRootCmd(loader, secrets)
	rootCmd.SetContext(context.Background())

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	fullArgs := append([]string{"--base-url", serverURL, "--token", "tok_test"}, args...)
	rootCmd.SetArgs(fullArgs)

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// writeMeTestJSONFile creates a temp JSON file.
func writeMeTestJSONFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/input.json"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Tests: me get
// ---------------------------------------------------------------------------

func TestMe_Get_JSON(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["id"] != "usr_001" {
		t.Errorf("expected id usr_001, got %v", result["id"])
	}
	if result["email"] != "user@example.com" {
		t.Errorf("expected email user@example.com, got %v", result["email"])
	}
}

func TestMe_Get_Table(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "me", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "usr_001") {
		t.Error("expected user ID in table output")
	}
	if !strings.Contains(stdout, "Test User") {
		t.Error("expected display name in table output")
	}
}

// ---------------------------------------------------------------------------
// Tests: me notification-prefs get
// ---------------------------------------------------------------------------

func TestMe_NotificationPrefs_Get_JSON(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "notification-prefs", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	// Check a known boolean field
	if result["email_notifications"] != true {
		t.Errorf("expected email_notifications true, got %v", result["email_notifications"])
	}
}

func TestMe_NotificationPrefs_Get_Table(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "me", "notification-prefs", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Email Notifications") || !strings.Contains(stdout, "Push Notifications") {
		t.Error("expected notification preference fields in table output")
	}
}

// ---------------------------------------------------------------------------
// Tests: me notification-prefs update
// ---------------------------------------------------------------------------

func TestMe_NotificationPrefs_Update(t *testing.T) {
	server := newMeStubServer(t)

	tmpFile := writeMeTestJSONFile(t, `{"push_notifications": true}`)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "notification-prefs", "update", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["push_notifications"] != true {
		t.Errorf("expected push_notifications true after update, got %v", result["push_notifications"])
	}
}

func TestMe_NotificationPrefs_Update_MissingFile(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "me", "notification-prefs", "update")
	if err == nil {
		t.Fatal("expected error when --from-file not provided")
	}
}

// ---------------------------------------------------------------------------
// Tests: me daily-recap-prefs get
// ---------------------------------------------------------------------------

func TestMe_DailyRecapPrefs_Get_JSON(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "daily-recap-prefs", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["timezone"] != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %v", result["timezone"])
	}
}

func TestMe_DailyRecapPrefs_Get_Table(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "me", "daily-recap-prefs", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Timezone") {
		t.Error("expected Timezone field in table output")
	}
}

// ---------------------------------------------------------------------------
// Tests: me daily-recap-prefs update
// ---------------------------------------------------------------------------

func TestMe_DailyRecapPrefs_Update(t *testing.T) {
	server := newMeStubServer(t)

	tmpFile := writeMeTestJSONFile(t, `{"timezone": "UTC", "send_time": "08:00"}`)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "daily-recap-prefs", "update", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["timezone"] != "UTC" {
		t.Errorf("expected timezone UTC after update, got %v", result["timezone"])
	}
}

// ---------------------------------------------------------------------------
// Tests: me avatar upload-url
// ---------------------------------------------------------------------------

func TestMe_Avatar_UploadURL(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "avatar", "upload-url", "--content-type", "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["upload_url"] == nil || result["upload_url"] == "" {
		t.Error("expected upload_url in response")
	}
}

func TestMe_Avatar_UploadURL_MissingContentType(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "me", "avatar", "upload-url")
	if err == nil {
		t.Fatal("expected error when --content-type not provided")
	}
}

// ---------------------------------------------------------------------------
// Tests: me avatar confirm
// ---------------------------------------------------------------------------

func TestMe_Avatar_Confirm(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "me", "avatar", "confirm", "--asset-url", "https://cdn.example.com/avatars/usr_001.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["avatar_url"] == nil || result["avatar_url"] == "" {
		t.Error("expected avatar_url in response")
	}
}

func TestMe_Avatar_Confirm_MissingAssetURL(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "me", "avatar", "confirm")
	if err == nil {
		t.Fatal("expected error when --asset-url not provided")
	}
}

// ---------------------------------------------------------------------------
// Tests: me avatar delete (DESTRUCTIVE)
// ---------------------------------------------------------------------------

func TestMe_Avatar_Delete_WithYes(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "--yes", "me", "avatar", "delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMe_Avatar_Delete_NonTTY_WithoutYes(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "me", "avatar", "delete")
	if err == nil {
		t.Fatal("expected error for non-TTY destructive delete without --yes")
	}
}

// ---------------------------------------------------------------------------
// Tests: me subcommand tree
// ---------------------------------------------------------------------------

func TestMe_HelpShown(t *testing.T) {
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default"},
		},
	}

	rootCmd := NewRootCmd(loader, nil)
	rootCmd.SetContext(context.Background())
	outBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)

	rootCmd.SetArgs([]string{"me", "--help"})
	_ = rootCmd.Execute()

	helpOutput := outBuf.String()
	if !strings.Contains(helpOutput, "get") {
		t.Error("expected 'get' in me help")
	}
	if !strings.Contains(helpOutput, "notification-prefs") {
		t.Error("expected 'notification-prefs' in me help")
	}
	if !strings.Contains(helpOutput, "avatar") {
		t.Error("expected 'avatar' in me help")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests for me.go
// ---------------------------------------------------------------------------

func TestMe_NotificationPrefs_Update_Table(t *testing.T) {
	server := newMeStubServer(t)

	tmpFile := writeMeTestJSONFile(t, `{"push_notifications": true}`)

	stdout, _, err := runMeCmd(t, server.URL, "me", "notification-prefs", "update", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Push Notifications") {
		t.Error("expected Push Notifications in table output")
	}
}

func TestMe_DailyRecapPrefs_Update_Table(t *testing.T) {
	server := newMeStubServer(t)

	tmpFile := writeMeTestJSONFile(t, `{"timezone": "UTC"}`)

	stdout, _, err := runMeCmd(t, server.URL, "me", "daily-recap-prefs", "update", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Timezone") {
		t.Error("expected Timezone in table output")
	}
}

func TestMe_DailyRecapPrefs_Update_MissingFile(t *testing.T) {
	server := newMeStubServer(t)

	_, _, err := runMeCmd(t, server.URL, "me", "daily-recap-prefs", "update")
	if err == nil {
		t.Fatal("expected error when --from-file not provided")
	}
}

func TestMe_Avatar_UploadURL_Table(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "me", "avatar", "upload-url", "--content-type", "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Upload URL") {
		t.Error("expected Upload URL in table output")
	}
}

func TestMe_Avatar_Confirm_Table(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "me", "avatar", "confirm", "--asset-url", "https://cdn.example.com/avatars/usr_001.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Avatar URL") {
		t.Error("expected Avatar URL in table output")
	}
}

func TestMe_Avatar_Delete_JSONPlan(t *testing.T) {
	server := newMeStubServer(t)

	stdout, _, err := runMeCmd(t, server.URL, "--output", "json", "--yes", "me", "avatar", "delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "plan") {
		t.Error("expected JSON plan in output")
	}
	if !strings.Contains(stdout, "Delete avatar") {
		t.Error("expected 'Delete avatar' operation in plan")
	}
}

// Ensure imports are used.
var _ = io.Discard
