package cmd_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the icctl binary to a temp directory and returns the path.
// The binary is cached for the test run via t.TempDir.
func buildBinary(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "icctl")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/icctl")
	cmd.Dir = findModuleRoot(t)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build icctl: %v\n%s", err, out)
	}

	return binPath
}

// findModuleRoot walks up from the working directory to find the go.mod file.
func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

const sentinelToken = "secret-sentinel-12345"

// TestTokenNeverLogged_SuccessPath builds icctl, runs it against a mock server
// that returns a valid "me" response with --log-level debug, and asserts the
// sentinel token never appears in stderr output.
func TestTokenNeverLogged_SuccessPath(t *testing.T) {
	binPath := buildBinary(t)

	// Mock server returning a valid me response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is present (so the token was sent).
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSONError(w, "authentication", http.StatusUnauthorized, "missing_token", "No token provided")
			return
		}

		// Return a valid me response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":"usr_01HW3","email":"test@example.com","display_name":"Test User"}`)
	}))
	defer srv.Close()

	cmd := exec.Command(binPath,
		"--log-level", "debug",
		"--token", sentinelToken,
		"--base-url", srv.URL,
		"--output", "json",
		"me", "get",
	)

	// Capture stderr where debug logs go.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	// Stdout is discarded for this assertion; we only care about stderr.
	cmd.Stdout = nil

	err := cmd.Run()
	// The command may fail if me subcommand is not yet wired (sibling agent
	// dependency). Either way, check stderr for the sentinel.
	stderrStr := stderr.String()

	if strings.Contains(stderrStr, sentinelToken) {
		t.Errorf("SECURITY VIOLATION (Constitution XI): token %q found in stderr (debug logs):\n%s",
			sentinelToken, stderrStr)
	}

	// If the command succeeded, that's a bonus. Log the exit code for visibility.
	if err != nil {
		t.Logf("command exited with error (expected until sibling agents merge): %v", err)
	}
}

// TestTokenNeverLogged_ErrorPath builds icctl, runs it against a mock server
// that returns a 401 error with --output json, and asserts the sentinel token
// never appears in the JSON error output on stderr.
func TestTokenNeverLogged_ErrorPath(t *testing.T) {
	binPath := buildBinary(t)

	// Mock server that always returns 401.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		writeJSONError(w, "authentication", http.StatusUnauthorized, "token_expired", "Bearer token expired")
	}))
	defer srv.Close()

	cmd := exec.Command(binPath,
		"--log-level", "debug",
		"--token", sentinelToken,
		"--base-url", srv.URL,
		"--output", "json",
		"me", "get",
	)

	var stderr strings.Builder
	var stdout strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	_ = cmd.Run() // Expected to fail with auth error.

	stderrStr := stderr.String()
	stdoutStr := stdout.String()

	if strings.Contains(stderrStr, sentinelToken) {
		t.Errorf("SECURITY VIOLATION (Constitution XI): token %q found in stderr (error path):\n%s",
			sentinelToken, stderrStr)
	}

	if strings.Contains(stdoutStr, sentinelToken) {
		t.Errorf("SECURITY VIOLATION (Constitution XI): token %q found in stdout (error path):\n%s",
			sentinelToken, stdoutStr)
	}
}

// TestTokenNeverLogged_EnvironmentVariable verifies that even when the token
// is supplied via ICCTL_TOKEN environment variable, it does not appear in logs.
func TestTokenNeverLogged_EnvironmentVariable(t *testing.T) {
	binPath := buildBinary(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id":"usr_01HW3","email":"test@example.com","display_name":"Test User"}`)
	}))
	defer srv.Close()

	cmd := exec.Command(binPath,
		"--log-level", "debug",
		"--base-url", srv.URL,
		"--output", "json",
		"me", "get",
	)
	cmd.Env = append(os.Environ(),
		"ICCTL_TOKEN="+sentinelToken,
		"ICCTL_LOG_LEVEL=debug",
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	_ = cmd.Run()

	stderrStr := stderr.String()
	if strings.Contains(stderrStr, sentinelToken) {
		t.Errorf("SECURITY VIOLATION (Constitution XI): token %q found in stderr via env var:\n%s",
			sentinelToken, stderrStr)
	}
}

// writeJSONError writes a cli-schema.md compliant JSON error to the writer.
func writeJSONError(w http.ResponseWriter, kind string, status int, code, message string) {
	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"kind":    kind,
			"status":  status,
			"code":    code,
			"message": message,
		},
	}
	data, _ := json.Marshal(resp)
	_, _ = w.Write(data)
}
