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

func TestConnectorsListByLocationCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/locations/loc_123/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":                "con_001",
					"name":              "Google Business Profile",
					"type":              "google_business_profile",
					"type_display_name": "Google Business Profile",
					"status":            "active",
					"status_display":    "Active",
					"created_at":        "2026-01-15T10:00:00Z",
					"updated_at":        "2026-04-01T12:00:00Z",
					"last_synced_at":    "2026-04-10T08:00:00Z",
				},
				{
					"id":                "con_002",
					"name":              "Facebook",
					"type":              "facebook",
					"type_display_name": "Facebook",
					"status":            "error",
					"status_display":    "Error",
					"status_message":    "Auth token expired",
					"created_at":        "2026-02-01T10:00:00Z",
					"updated_at":        "2026-04-05T12:00:00Z",
					"last_synced_at":    nil,
				},
			},
			"pagination": map[string]interface{}{
				"has_more": false,
			},
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
		"connectors", "list-by-location", "loc_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "con_001") {
		t.Errorf("expected 'con_001' in output, got: %s", output)
	}
	if !strings.Contains(output, "Google Business Profile") {
		t.Errorf("expected 'Google Business Profile' in output, got: %s", output)
	}
	if !strings.Contains(output, "con_002") {
		t.Errorf("expected 'con_002' in output, got: %s", output)
	}
}

func TestConnectorsListByLocationCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":                "con_001",
					"name":              "Google Business Profile",
					"type":              "google_business_profile",
					"type_display_name": "Google Business Profile",
					"status":            "active",
					"status_display":    "Active",
					"created_at":        "2026-01-15T10:00:00Z",
					"updated_at":        "2026-04-01T12:00:00Z",
					"last_synced_at":    "2026-04-10T08:00:00Z",
				},
			},
			"pagination": map[string]interface{}{
				"has_more": false,
			},
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
		"connectors", "list-by-location", "loc_123",
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
}

func TestConnectorsListByLocationCmd_Pagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp map[string]interface{}
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			resp = map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":                "con_001",
						"name":              "Connector Page 1",
						"type":              "google",
						"type_display_name": "Google",
						"status":            "active",
						"status_display":    "Active",
						"created_at":        "2026-01-15T10:00:00Z",
						"updated_at":        "2026-04-01T12:00:00Z",
					},
				},
				"pagination": map[string]interface{}{
					"has_more":    true,
					"next_cursor": "page2token",
				},
			}
		} else if cursor == "page2token" {
			resp = map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":                "con_002",
						"name":              "Connector Page 2",
						"type":              "facebook",
						"type_display_name": "Facebook",
						"status":            "active",
						"status_display":    "Active",
						"created_at":        "2026-02-01T10:00:00Z",
						"updated_at":        "2026-04-05T12:00:00Z",
					},
				},
				"pagination": map[string]interface{}{
					"has_more": false,
				},
			}
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
		"connectors", "list-by-location", "loc_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "con_001") {
		t.Errorf("expected 'con_001' from page 1, got: %s", output)
	}
	if !strings.Contains(output, "con_002") {
		t.Errorf("expected 'con_002' from page 2, got: %s", output)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestConnectorsListByCompanyCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/companies/cmp_123/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":                "con_001",
					"name":              "Google Business Profile",
					"type":              "google_business_profile",
					"type_display_name": "Google Business Profile",
					"status":            "active",
					"status_display":    "Active",
					"location_id":       "loc_001",
					"location_name":     "Downtown Office",
					"created_at":        "2026-01-15T10:00:00Z",
					"updated_at":        "2026-04-01T12:00:00Z",
					"last_synced_at":    "2026-04-10T08:00:00Z",
				},
			},
			"errors": []map[string]string{
				{"name": "Broken Connector", "status_message": "Auth expired"},
			},
			"pagination": map[string]interface{}{
				"has_more": false,
			},
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
		"connectors", "list-by-company", "cmp_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "con_001") {
		t.Errorf("expected 'con_001' in output, got: %s", output)
	}
	if !strings.Contains(output, "Downtown Office") {
		t.Errorf("expected 'Downtown Office' in output, got: %s", output)
	}
	if !strings.Contains(output, "Broken Connector") {
		t.Errorf("expected error summary 'Broken Connector' in output, got: %s", output)
	}
	if !strings.Contains(output, "Auth expired") {
		t.Errorf("expected error summary 'Auth expired' in output, got: %s", output)
	}
}

func TestConnectorsListByCompanyCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":                "con_001",
					"name":              "Google Business Profile",
					"type":              "google_business_profile",
					"type_display_name": "Google Business Profile",
					"status":            "active",
					"status_display":    "Active",
					"location_id":       "loc_001",
					"location_name":     "Downtown Office",
					"created_at":        "2026-01-15T10:00:00Z",
					"updated_at":        "2026-04-01T12:00:00Z",
					"last_synced_at":    "2026-04-10T08:00:00Z",
				},
			},
			"errors": []map[string]string{
				{"name": "Broken Connector", "status_message": "Auth expired"},
			},
			"pagination": map[string]interface{}{
				"has_more": false,
			},
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
		"connectors", "list-by-company", "cmp_123",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Items        []interface{} `json:"items"`
		ErrorSummary []struct {
			Name          string `json:"name"`
			StatusMessage string `json:"status_message"`
		} `json:"error_summary"`
		PageInfo struct {
			ItemsFetched int `json:"items_fetched"`
		} `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if len(result.ErrorSummary) != 1 {
		t.Errorf("expected 1 error summary item, got %d", len(result.ErrorSummary))
	} else if result.ErrorSummary[0].Name != "Broken Connector" {
		t.Errorf("expected error name 'Broken Connector', got %q", result.ErrorSummary[0].Name)
	}
}

func TestConnectorsListByLocationCmd_MissingArg(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"connectors", "list-by-location"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing location ID argument")
	}
}

func TestConnectorsListByCompanyCmd_MissingArg(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"connectors", "list-by-company"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing company ID argument")
	}
}

func TestConnectorsCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"connectors", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
