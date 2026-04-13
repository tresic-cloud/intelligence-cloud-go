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

func TestProductsListCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"data": []map[string]string{
				{"key": "listings", "name": "Listings Management"},
				{"key": "social", "name": "Social Media"},
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
		"products", "list",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "listings") {
		t.Errorf("expected 'listings' in table output, got: %s", output)
	}
	if !strings.Contains(output, "Listings Management") {
		t.Errorf("expected 'Listings Management' in table output, got: %s", output)
	}
	if !strings.Contains(output, "social") {
		t.Errorf("expected 'social' in table output, got: %s", output)
	}
}

func TestProductsListCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]string{
				{"key": "listings", "name": "Listings Management"},
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
		"products", "list",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Items    []map[string]interface{} `json:"items"`
		PageInfo struct {
			ItemsFetched int `json:"items_fetched"`
			PagesFetched int `json:"pages_fetched"`
		} `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, buf.String())
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.PageInfo.ItemsFetched != 1 {
		t.Errorf("expected items_fetched=1, got %d", result.PageInfo.ItemsFetched)
	}
	if result.PageInfo.PagesFetched != 1 {
		t.Errorf("expected pages_fetched=1, got %d", result.PageInfo.PagesFetched)
	}
}

func TestProductsListCmd_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []interface{}{},
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
		"products", "list",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Items []interface{} `json:"items"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestProductsCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"products", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsListCmd_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "token_expired",
			"message": "Bearer token expired",
		})
	}))
	defer srv.Close()

	root := NewRootCmd(nil, nil)
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"--base-url", srv.URL,
		"--token", "bad-token",
		"products", "list",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
