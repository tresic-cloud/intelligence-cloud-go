package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsersPatchCompanyCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/companies/cmp_123/users/usr_456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if reqBody["first_name"] != "Jane" {
			t.Errorf("expected first_name=Jane, got %v", reqBody["first_name"])
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "usr_456",
				"email":      "jane@example.com",
				"first_name": "Jane",
				"last_name":  "Doe",
				"role":       "admin",
				"status":     "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "update.json")
	err := os.WriteFile(jsonFile, []byte(`{"first_name":"Jane","last_name":"Doe"}`), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "test-token",
		"--output", "table",
		"users", "patch-company", "cmp_123", "usr_456",
		"--from-file", jsonFile,
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "usr_456") {
		t.Errorf("expected 'usr_456' in output, got: %s", output)
	}
	if !strings.Contains(output, "Jane") {
		t.Errorf("expected 'Jane' in output, got: %s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("expected 'admin' in output, got: %s", output)
	}
}

func TestUsersPatchCompanyCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "usr_456",
				"email":      "jane@example.com",
				"first_name": "Jane",
				"last_name":  "Doe",
				"role":       "admin",
				"status":     "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "update.json")
	err := os.WriteFile(jsonFile, []byte(`{"first_name":"Jane"}`), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "test-token",
		"--output", "json",
		"users", "patch-company", "cmp_123", "usr_456",
		"--from-file", jsonFile,
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}
	if result["id"] != "usr_456" {
		t.Errorf("expected id=usr_456, got %v", result["id"])
	}
}

func TestUsersPatchCompanyCmd_MissingArgs(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"users", "patch-company"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing arguments")
	}
}

func TestUsersPatchCompanyCmd_MissingFromFile(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"users", "patch-company", "cmp_123", "usr_456"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing --from-file")
	}
}

func TestUsersPatchCompanyCmd_InvalidJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "bad.json")
	err := os.WriteFile(jsonFile, []byte(`{invalid json`), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", "http://127.0.0.1:9999",
		"--token", "test-token",
		"users", "patch-company", "cmp_123", "usr_456",
		"--from-file", jsonFile,
	})

	execErr := root.Execute()
	if execErr == nil {
		t.Fatal("expected error for invalid JSON file")
	}
	if !strings.Contains(execErr.Error(), "parse JSON") {
		t.Errorf("expected parse JSON error, got: %v", execErr)
	}
}

func TestUsersPatchCompanyCmd_NonexistentFile(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", "http://127.0.0.1:9999",
		"--token", "test-token",
		"users", "patch-company", "cmp_123", "usr_456",
		"--from-file", "/nonexistent/file.json",
	})

	execErr := root.Execute()
	if execErr == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestUsersCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"users", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
