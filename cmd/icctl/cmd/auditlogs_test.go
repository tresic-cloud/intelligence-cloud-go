package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuditLogsListCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/resellers/rsl_123/audit-logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "log_001",
					"action":     "user.login",
					"actor_id":   "usr_100",
					"actor_name": "Alice Smith",
					"details":    "User logged in via SSO",
					"timestamp":  "2026-04-10T08:30:00Z",
				},
				{
					"id":         "log_002",
					"action":     "reseller.update",
					"actor_id":   "usr_200",
					"actor_name": "Bob Jones",
					"details":    "Updated reseller settings",
					"timestamp":  "2026-04-11T14:00:00Z",
				},
			},
			"has_more": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "test-token",
		"--output", "table",
		"audit-logs", "list", "--reseller", "rsl_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "log_001") {
		t.Errorf("expected 'log_001' in output, got: %s", output)
	}
	if !strings.Contains(output, "user.login") {
		t.Errorf("expected 'user.login' in output, got: %s", output)
	}
	if !strings.Contains(output, "Alice Smith") {
		t.Errorf("expected 'Alice Smith' in output, got: %s", output)
	}
	if !strings.Contains(output, "Bob Jones") {
		t.Errorf("expected 'Bob Jones' in output, got: %s", output)
	}
}

func TestAuditLogsListCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "log_001",
					"action":     "user.login",
					"actor_id":   "usr_100",
					"actor_name": "Alice Smith",
					"details":    "User logged in",
					"timestamp":  "2026-04-10T08:30:00Z",
				},
			},
			"has_more": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "test-token",
		"--output", "json",
		"audit-logs", "list", "--reseller", "rsl_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Items    []map[string]interface{} `json:"items"`
		PageInfo struct {
			ItemsFetched int  `json:"items_fetched"`
			PagesFetched int  `json:"pages_fetched"`
			HasNextPage  bool `json:"has_next_page"`
		} `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.PageInfo.ItemsFetched != 1 {
		t.Errorf("expected items_fetched=1, got %d", result.PageInfo.ItemsFetched)
	}
}

func TestAuditLogsListCmd_WithFilters(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		resp := map[string]interface{}{
			"data":     []interface{}{},
			"has_more": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "test-token",
		"--output", "json",
		"audit-logs", "list",
		"--reseller", "rsl_123",
		"--action", "user.login",
		"--since", "2026-01-01",
		"--until", "2026-04-01",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedQuery, "action=user.login") {
		t.Errorf("expected action filter in query, got: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "start_date=2026-01-01") {
		t.Errorf("expected start_date filter in query, got: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "end_date=2026-04-01") {
		t.Errorf("expected end_date filter in query, got: %s", capturedQuery)
	}
}

func TestAuditLogsListCmd_MissingReseller(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"audit-logs", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing --reseller flag")
	}
}

func TestAuditLogsCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"audit-logs", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
