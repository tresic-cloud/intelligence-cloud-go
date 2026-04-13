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
// Helpers: stub API server for resellers + audit-logs
// ---------------------------------------------------------------------------

// newResellerStubServer returns an httptest.Server that stubs all reseller and
// audit-log endpoints used by the CLI subcommands.
func newResellerStubServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// GET /api/v1/resellers — list
	mux.HandleFunc("GET /api/v1/resellers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":            "rsl_001",
					"reseller_name": "Acme Reseller",
					"is_active":     true,
					"created_at":    "2026-01-01T00:00:00Z",
				},
				{
					"id":            "rsl_002",
					"reseller_name": "Beta Reseller",
					"is_active":     false,
					"created_at":    "2026-02-01T00:00:00Z",
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// GET /api/v1/resellers/{id} — get
	mux.HandleFunc("GET /api/v1/resellers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "rsl_notfound" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "not_found",
					"message": "reseller not found",
				},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                    id,
			"reseller_name":         "Acme Reseller",
			"is_active":             true,
			"primary_contact_name":  "Jane Doe",
			"primary_contact_email": "jane@acme.com",
			"primary_contact_phone": "555-0100",
			"created_at":            "2026-01-01T00:00:00Z",
		})
	})

	// POST /api/v1/resellers — create
	mux.HandleFunc("POST /api/v1/resellers", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":            "rsl_new",
			"reseller_name": req["reseller_name"],
			"is_active":     true,
			"created_at":    "2026-04-01T00:00:00Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// PUT /api/v1/resellers/{id} — update
	mux.HandleFunc("PUT /api/v1/resellers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"id":            id,
			"reseller_name": req["reseller_name"],
			"is_active":     true,
			"updated_at":    "2026-04-02T00:00:00Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// PATCH /api/v1/resellers/{id} — patch
	mux.HandleFunc("PATCH /api/v1/resellers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"reseller_name": "Patched Reseller",
			"is_active":     true,
			"updated_at":    "2026-04-03T00:00:00Z",
		})
	})

	// POST /api/v1/resellers/{id}/deactivate — deactivate
	mux.HandleFunc("POST /api/v1/resellers/{id}/deactivate", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"reseller_name": "Acme Reseller",
			"is_active":     false,
		})
	})

	// POST /api/v1/resellers/{id}/reactivate — reactivate
	mux.HandleFunc("POST /api/v1/resellers/{id}/reactivate", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"reseller_name": "Acme Reseller",
			"is_active":     true,
		})
	})

	// GET /api/v1/resellers/{id}/audit-logs — audit-logs
	mux.HandleFunc("GET /api/v1/resellers/{id}/audit-logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "al_001",
					"action":     "reseller.created",
					"actor_id":   "usr_001",
					"actor_name": "Admin User",
					"details":    "Reseller created",
					"timestamp":  "2026-01-01T00:00:00Z",
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// runResellersCmd executes a command against a stub server with the given args.
// It returns stdout, stderr, and any execution error.
func runResellersCmd(t *testing.T, serverURL string, args ...string) (stdout, stderr string, err error) {
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
			"default": "tok_test_resellers",
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

// writeTestJSONFile creates a temp JSON file for test input.
func writeTestJSONFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/input.json"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Tests: resellers list
// ---------------------------------------------------------------------------

func TestResellers_List_Table(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "resellers", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "rsl_001") {
		t.Error("expected reseller rsl_001 in table output")
	}
	if !strings.Contains(stdout, "Acme Reseller") {
		t.Error("expected reseller name 'Acme Reseller' in table output")
	}
	if !strings.Contains(stdout, "rsl_002") {
		t.Error("expected reseller rsl_002 in table output")
	}
}

func TestResellers_List_JSON(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, stdout)
	}

	if _, ok := result["items"]; !ok {
		t.Error("expected 'items' key in JSON output")
	}
	if _, ok := result["page_info"]; !ok {
		t.Error("expected 'page_info' key in JSON output")
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(result["items"], &items); err != nil {
		t.Fatalf("failed to unmarshal items: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestResellers_List_WithPageSize(t *testing.T) {
	server := newResellerStubServer(t)

	// Ensure --page-size flag is accepted without error
	_, _, err := runResellersCmd(t, server.URL, "resellers", "list", "--page-size", "10")
	if err != nil {
		t.Fatalf("unexpected error with --page-size: %v", err)
	}
}

func TestResellers_List_WithMaxItems(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "list", "--max-items", "1")
	if err != nil {
		t.Fatalf("unexpected error with --max-items: %v", err)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item with --max-items 1, got %d", len(result.Items))
	}
}

func TestResellers_List_WithFilter(t *testing.T) {
	server := newResellerStubServer(t)

	// Ensure --filter flag is accepted
	_, _, err := runResellersCmd(t, server.URL, "resellers", "list", "--filter", "country=US")
	if err != nil {
		t.Fatalf("unexpected error with --filter: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers get
// ---------------------------------------------------------------------------

func TestResellers_Get_JSON(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "get", "rsl_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["id"] != "rsl_001" {
		t.Errorf("expected id rsl_001, got %v", result["id"])
	}
}

func TestResellers_Get_MissingID(t *testing.T) {
	server := newResellerStubServer(t)

	_, _, err := runResellersCmd(t, server.URL, "resellers", "get")
	if err == nil {
		t.Fatal("expected error for missing ID argument")
	}
}

func TestResellers_Get_Table(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "resellers", "get", "rsl_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "rsl_001") {
		t.Error("expected reseller ID in table output")
	}
	if !strings.Contains(stdout, "Acme Reseller") {
		t.Error("expected reseller name in table output")
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers create
// ---------------------------------------------------------------------------

func TestResellers_Create_FromFile(t *testing.T) {
	server := newResellerStubServer(t)

	tmpFile := writeTestJSONFile(t, `{
		"reseller_name": "New Reseller",
		"primary_contact_name": "John",
		"primary_contact_email": "john@example.com",
		"primary_contact_phone": "555-0123"
	}`)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "create", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["id"] != "rsl_new" {
		t.Errorf("expected id rsl_new, got %v", result["id"])
	}
}

func TestResellers_Create_MissingInput(t *testing.T) {
	server := newResellerStubServer(t)

	_, _, err := runResellersCmd(t, server.URL, "resellers", "create")
	if err == nil {
		t.Fatal("expected error when no input provided")
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers update
// ---------------------------------------------------------------------------

func TestResellers_Update_FromFile(t *testing.T) {
	server := newResellerStubServer(t)

	tmpFile := writeTestJSONFile(t, `{
		"reseller_name": "Updated Reseller",
		"primary_contact_name": "Updated",
		"primary_contact_email": "updated@example.com",
		"primary_contact_phone": "555-0999"
	}`)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "update", "rsl_001", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["id"] != "rsl_001" {
		t.Errorf("expected id rsl_001, got %v", result["id"])
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers patch
// ---------------------------------------------------------------------------

func TestResellers_Patch_FromFile(t *testing.T) {
	server := newResellerStubServer(t)

	tmpFile := writeTestJSONFile(t, `{"reseller_name": "Patched Reseller"}`)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "patch", "rsl_001", "--from-file", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers deactivate (DESTRUCTIVE)
// ---------------------------------------------------------------------------

func TestResellers_Deactivate_WithYes(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "--yes", "resellers", "deactivate", "rsl_001", "--reason", "test deactivation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output contains the plan then the result. Parse the last JSON object.
	lines := strings.TrimSpace(stdout)
	// There should be at least the plan and the result
	if !strings.Contains(lines, "plan") {
		t.Error("expected JSON plan in output")
	}
}

func TestResellers_Deactivate_NonTTY_WithoutYes_Exits2(t *testing.T) {
	server := newResellerStubServer(t)

	// Non-TTY: no --yes flag, stdin is not a terminal.
	// The command should refuse with an error that maps to exit code 2.
	_, _, err := runResellersCmd(t, server.URL, "resellers", "deactivate", "rsl_001", "--reason", "test")
	if err == nil {
		t.Fatal("expected error for non-TTY destructive operation without --yes")
	}

	// Verify the error message indicates non-TTY refusal.
	if !strings.Contains(err.Error(), "not a TTY") && !strings.Contains(err.Error(), "refusing") {
		t.Errorf("expected non-TTY refusal error, got: %v", err)
	}
}

func TestResellers_Deactivate_MissingReason(t *testing.T) {
	server := newResellerStubServer(t)

	_, _, err := runResellersCmd(t, server.URL, "--yes", "resellers", "deactivate", "rsl_001")
	if err == nil {
		t.Fatal("expected error for missing --reason flag")
	}
}

func TestResellers_Deactivate_JSONPlan_BeforePrompt(t *testing.T) {
	server := newResellerStubServer(t)

	// With --yes + --output json, a plan should be emitted to stdout
	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "--yes", "resellers", "deactivate", "rsl_001", "--reason", "plan test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The plan should contain the destructive preview
	if !strings.Contains(stdout, "plan") {
		t.Error("expected JSON plan in output before API call")
	}
	if !strings.Contains(stdout, "Deactivate reseller") {
		t.Error("expected 'Deactivate reseller' operation in plan")
	}
}

func TestResellers_Deactivate_TTYPrompt_Accepted(t *testing.T) {
	server := newResellerStubServer(t)

	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default", Environment: "dev", BaseURL: server.URL},
		},
	}
	secrets := &mockSecretLoader{
		tokens: map[string]string{"default": "tok_test"},
	}

	rootCmd := NewRootCmd(loader, secrets)
	rootCmd.SetContext(context.Background())

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	// Inject "y\n" into stdin to simulate user confirmation
	rootCmd.SetIn(strings.NewReader("y\n"))

	rootCmd.SetArgs([]string{
		"--base-url", server.URL,
		"--token", "tok_test",
		"resellers", "deactivate", "rsl_001",
		"--reason", "tty test",
		"--force-tty",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The stderr should contain the preview
	if !strings.Contains(errBuf.String(), "Deactivate reseller") {
		t.Error("expected destructive preview in stderr")
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers reactivate (NOT destructive)
// ---------------------------------------------------------------------------

func TestResellers_Reactivate_JSON(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "reactivate", "rsl_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if result["id"] != "rsl_001" {
		t.Errorf("expected id rsl_001, got %v", result["id"])
	}
}

func TestResellers_Reactivate_NoConfirmation(t *testing.T) {
	// Reactivate is NOT destructive — should not require --yes
	server := newResellerStubServer(t)

	_, stderrStr, err := runResellersCmd(t, server.URL, "resellers", "reactivate", "rsl_001")
	if err != nil {
		t.Fatalf("reactivate should not require confirmation: %v", err)
	}

	// Should NOT contain destructive preview
	if strings.Contains(stderrStr, "operation is about to be performed") {
		t.Error("reactivate should NOT show destructive preview")
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers audit-logs
// ---------------------------------------------------------------------------

func TestResellers_AuditLogs_JSON(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json", "resellers", "audit-logs", "rsl_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if _, ok := result["items"]; !ok {
		t.Error("expected 'items' key in audit-logs JSON output")
	}
}

func TestResellers_AuditLogs_WithDateFilters(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "--output", "json",
		"resellers", "audit-logs", "rsl_001",
		"--since", "2026-01-01",
		"--until", "2026-04-01",
	)
	if err != nil {
		t.Fatalf("unexpected error with date filters: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}
}

func TestResellers_AuditLogs_Table(t *testing.T) {
	server := newResellerStubServer(t)

	stdout, _, err := runResellersCmd(t, server.URL, "resellers", "audit-logs", "rsl_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "reseller.created") {
		t.Error("expected audit action in table output")
	}
}

func TestResellers_AuditLogs_AllFilters(t *testing.T) {
	server := newResellerStubServer(t)

	_, _, err := runResellersCmd(t, server.URL, "resellers", "audit-logs", "rsl_001",
		"--action", "reseller.created",
		"--since", "2026-01-01",
		"--until", "2026-04-01",
		"--actor-id", "usr_001",
		"--search", "created",
	)
	if err != nil {
		t.Fatalf("unexpected error with all filters: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: resellers subcommand tree
// ---------------------------------------------------------------------------

func TestResellers_HelpShown(t *testing.T) {
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

	rootCmd.SetArgs([]string{"resellers", "--help"})
	_ = rootCmd.Execute()

	output := outBuf.String()
	if !strings.Contains(output, "list") {
		t.Error("expected 'list' in resellers help")
	}
	if !strings.Contains(output, "get") {
		t.Error("expected 'get' in resellers help")
	}
	if !strings.Contains(output, "create") {
		t.Error("expected 'create' in resellers help")
	}
}

func TestResellers_NoDeleteSubcommand(t *testing.T) {
	// Verify that "delete" is not a registered subcommand — the generated
	// client does not include DeleteReseller.
	loader := &mockProfileLoader{
		profiles: map[string]*ProfileConfig{
			"default": {Name: "default"},
		},
	}

	rootCmd := NewRootCmd(loader, nil)

	// Walk the resellers command tree and verify "delete" is absent.
	resellersCmd, _, err := rootCmd.Find([]string{"resellers"})
	if err != nil {
		t.Fatalf("failed to find resellers command: %v", err)
	}

	for _, sub := range resellersCmd.Commands() {
		if sub.Name() == "delete" {
			t.Fatal("resellers should NOT have a 'delete' subcommand (DeleteReseller is absent from generated client)")
		}
	}
}

// Ensure os import is used.
var _ = os.DevNull
