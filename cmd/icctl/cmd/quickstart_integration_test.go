//go:build integration

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

// TestQuickstartPartB reproduces specs/001-sdk-cli-foundation/quickstart.md
// Part B as an automated integration test. It exercises the full CLI binary
// against a mock httptest.Server, validating exit codes per cli-schema.md.
//
// This test is tagged //go:build integration and requires the full binary to
// build. It will FAIL until sibling agents (W4-1, W4-2) add their subcommands.
func TestQuickstartPartB(t *testing.T) {
	// Build the binary.
	binPath := buildIntegrationBinary(t)

	// Set up a temp config home for profile isolation.
	configHome := t.TempDir()

	// Start mock API server.
	srv := newQuickstartMockServer(t)
	defer srv.Close()

	// Helper to run icctl with common env.
	run := func(args ...string) (stdout, stderr string, exitCode int) {
		t.Helper()
		cmd := exec.Command(binPath, args...)
		cmd.Env = []string{
			"HOME=" + configHome,
			"XDG_CONFIG_HOME=" + configHome,
			"PATH=" + os.Getenv("PATH"),
		}

		var outBuf, errBuf strings.Builder
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		err := cmd.Run()
		exitCode = 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				t.Fatalf("unexpected exec error: %v", err)
			}
		}

		return outBuf.String(), errBuf.String(), exitCode
	}

	// runWithStdin runs icctl with data piped to stdin.
	runWithStdin := func(stdin string, args ...string) (stdout, stderr string, exitCode int) {
		t.Helper()
		cmd := exec.Command(binPath, args...)
		cmd.Env = []string{
			"HOME=" + configHome,
			"XDG_CONFIG_HOME=" + configHome,
			"PATH=" + os.Getenv("PATH"),
		}
		cmd.Stdin = strings.NewReader(stdin)

		var outBuf, errBuf strings.Builder
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		err := cmd.Run()
		exitCode = 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				t.Fatalf("unexpected exec error: %v", err)
			}
		}

		return outBuf.String(), errBuf.String(), exitCode
	}

	// Step 1: icctl version succeeds.
	t.Run("Step1_Version", func(t *testing.T) {
		stdout, stderr, code := run("version")
		if code != 0 {
			t.Fatalf("icctl version exited %d; stderr: %s", code, stderr)
		}
		if stdout == "" {
			t.Error("icctl version produced no stdout output")
		}
	})

	// Step 2: icctl profile add staging --environment staging --token-stdin
	t.Run("Step2_ProfileAddStaging", func(t *testing.T) {
		_, stderr, code := runWithStdin(
			"tok_staging_test_12345\n",
			"profile", "add", "staging",
			"--environment", "staging",
			"--base-url", srv.URL,
			"--token-stdin",
		)
		if code != 0 {
			t.Fatalf("profile add exited %d; stderr: %s", code, stderr)
		}
	})

	// Step 3: icctl profile test staging
	t.Run("Step3_ProfileTest", func(t *testing.T) {
		_, stderr, code := run(
			"--base-url", srv.URL,
			"profile", "test", "staging",
		)
		if code != 0 {
			t.Fatalf("profile test exited %d; stderr: %s", code, stderr)
		}
	})

	// Step 4: icctl --profile staging resellers list returns table output.
	t.Run("Step4_ResellersListTable", func(t *testing.T) {
		stdout, stderr, code := run(
			"--profile", "staging",
			"--base-url", srv.URL,
			"--token", "tok_staging_test_12345",
			"resellers", "list",
		)
		if code != 0 {
			t.Fatalf("resellers list (table) exited %d; stderr: %s", code, stderr)
		}
		if stdout == "" {
			t.Error("resellers list produced no stdout output")
		}
	})

	// Step 5: icctl --profile staging resellers list --output json returns valid JSON.
	t.Run("Step5_ResellersListJSON", func(t *testing.T) {
		stdout, stderr, code := run(
			"--profile", "staging",
			"--base-url", srv.URL,
			"--token", "tok_staging_test_12345",
			"resellers", "list",
			"--output", "json",
		)
		if code != 0 {
			t.Fatalf("resellers list (json) exited %d; stderr: %s", code, stderr)
		}
		if !json.Valid([]byte(stdout)) {
			t.Errorf("resellers list --output json did not produce valid JSON:\n%s", stdout)
		}
	})

	// Step 6: icctl --profile staging resellers deactivate <id> --yes executes
	// without an interactive prompt.
	t.Run("Step6_DestructiveWithYes", func(t *testing.T) {
		_, stderr, code := run(
			"--profile", "staging",
			"--base-url", srv.URL,
			"--token", "tok_staging_test_12345",
			"--yes",
			"resellers", "deactivate", "rsl_01HW3ABCDEF",
		)
		if code != 0 {
			t.Fatalf("resellers deactivate --yes exited %d; stderr: %s", code, stderr)
		}
	})

	// Step 7: icctl --profile staging resellers deactivate <id> without --yes
	// on non-TTY exits code 2 (cli-schema.md: non-TTY without --yes refused).
	t.Run("Step7_DestructiveNoYes_NonTTY_Exit2", func(t *testing.T) {
		_, _, code := run(
			"--profile", "staging",
			"--base-url", srv.URL,
			"--token", "tok_staging_test_12345",
			"resellers", "deactivate", "rsl_01HW3ABCDEF",
		)
		if code != 2 {
			t.Errorf("resellers deactivate without --yes on non-TTY: exit code = %d; want 2", code)
		}
	})

	// Step 8: Switch environments with profile add + set-default.
	t.Run("Step8_SwitchEnvironment", func(t *testing.T) {
		// Add prod profile.
		_, stderr, code := runWithStdin(
			"tok_prod_test_67890\n",
			"profile", "add", "prod",
			"--environment", "production",
			"--base-url", srv.URL,
			"--token-stdin",
		)
		if code != 0 {
			t.Fatalf("profile add prod exited %d; stderr: %s", code, stderr)
		}

		// Set default to prod.
		_, stderr, code = run("profile", "set-default", "prod")
		if code != 0 {
			t.Fatalf("profile set-default prod exited %d; stderr: %s", code, stderr)
		}

		// Verify the default profile is now prod by listing profiles.
		stdout, stderr, code := run("profile", "list")
		if code != 0 {
			t.Fatalf("profile list exited %d; stderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "prod") {
			t.Errorf("profile list does not contain 'prod': %s", stdout)
		}
	})
}

// buildIntegrationBinary builds the icctl binary for integration testing.
func buildIntegrationBinary(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "icctl")

	modRoot := findIntegrationModuleRoot(t)

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/icctl")
	cmd.Dir = modRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build icctl: %v\n%s", err, out)
	}

	return binPath
}

// findIntegrationModuleRoot walks up from the working directory to find go.mod.
func findIntegrationModuleRoot(t *testing.T) string {
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

// newQuickstartMockServer creates an httptest.Server that stubs IC API
// endpoints needed by the quickstart Part B walkthrough.
func newQuickstartMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// /me endpoint
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"id": "usr_01HW3XYZ",
			"email": "jason@tresic.cloud",
			"display_name": "Jason Goecke"
		}`)
	})

	// /resellers endpoint (list)
	mux.HandleFunc("/resellers", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{
				"items": [
					{
						"id": "rsl_01HW3ABCDEF",
						"name": "Acme Bakery, Inc.",
						"status": "active",
						"created_at": "2025-12-04T10:00:00Z"
					},
					{
						"id": "rsl_01HX7GHIJKL",
						"name": "Blue Harbor Pizza",
						"status": "active",
						"created_at": "2026-01-17T14:30:00Z"
					}
				],
				"page_info": {
					"items_fetched": 2,
					"pages_fetched": 1,
					"has_next_page": false,
					"next_page_token": ""
				}
			}`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /resellers/{id}/deactivate endpoint
	mux.HandleFunc("/resellers/rsl_01HW3ABCDEF/deactivate", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"id": "rsl_01HW3ABCDEF",
			"name": "Acme Bakery, Inc.",
			"status": "inactive"
		}`)
	})

	// /resellers/{id} endpoint (for fetching before destructive ops)
	mux.HandleFunc("/resellers/rsl_01HW3ABCDEF", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"id": "rsl_01HW3ABCDEF",
			"name": "Acme Bakery, Inc.",
			"status": "active",
			"created_at": "2025-12-04T10:00:00Z"
		}`)
	})

	return httptest.NewServer(mux)
}

// checkAuth validates the Authorization header and writes a 401 response if missing.
func checkAuth(w http.ResponseWriter, r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":{"kind":"authentication","status":401,"code":"missing_token","message":"No token provided"}}`)
		return false
	}
	return true
}
