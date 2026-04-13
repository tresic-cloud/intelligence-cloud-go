package intelligencecloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	genapi "github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

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

		resp := genapi.LocationConnectorListResponse{
			Data: []genapi.LocationConnectorItem{
				{
					Id:              "conn-1",
					Name:            "Google Reviews",
					Type:            genapi.LocationConnectorItemTypeGoogleReviews,
					TypeDisplayName: "Google Reviews",
					Status:          genapi.LocationConnectorItemStatusActive,
					StatusDisplay:   genapi.LocationConnectorItemStatusDisplayConnected,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
			Pagination: genapi.PaginationDTO{
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

		var resp genapi.LocationConnectorListResponse
		if page == 0 && cursor == "" {
			next := "loc-page2"
			resp = genapi.LocationConnectorListResponse{
				Data: []genapi.LocationConnectorItem{
					{Id: "c1", Name: "C1", Type: genapi.LocationConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.LocationConnectorItemStatusActive, StatusDisplay: genapi.LocationConnectorItemStatusDisplayConnected, CreatedAt: now, UpdatedAt: now},
				},
				Pagination: genapi.PaginationDTO{HasMore: true, NextCursor: &next},
			}
			page++
		} else if cursor == "loc-page2" {
			resp = genapi.LocationConnectorListResponse{
				Data: []genapi.LocationConnectorItem{
					{Id: "c2", Name: "C2", Type: genapi.LocationConnectorItemTypeNetsapiens, TypeDisplayName: "NetSapiens", Status: genapi.LocationConnectorItemStatusInactive, StatusDisplay: genapi.LocationConnectorItemStatusDisplayInactive, CreatedAt: now, UpdatedAt: now},
				},
				Pagination: genapi.PaginationDTO{HasMore: false, NextCursor: nil},
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

		resp := genapi.LocationConnectorListResponse{
			Data:       []genapi.LocationConnectorItem{},
			Pagination: genapi.PaginationDTO{HasMore: false},
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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

		resp := genapi.CompanyConnectorListResponse{
			Data: []genapi.CompanyConnectorItem{
				{
					Id:              "conn-c1",
					Name:            "Ooma VoIP",
					Type:            genapi.CompanyConnectorItemTypeOoma,
					TypeDisplayName: "Ooma",
					Status:          genapi.CompanyConnectorItemStatusActive,
					StatusDisplay:   genapi.CompanyConnectorItemStatusDisplayConnected,
					LocationId:      "loc-1",
					LocationName:    "Main Office",
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
			Errors: []genapi.ConnectorErrorSummaryItem{
				{Name: "Broken Connector", StatusMessage: "API key expired"},
			},
			Pagination: genapi.PaginationDTO{
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

		var resp genapi.CompanyConnectorListResponse

		if page == 0 && cursor == "" {
			next := "comp-page2"
			resp = genapi.CompanyConnectorListResponse{
				Data: []genapi.CompanyConnectorItem{
					{Id: "cc1", Name: "CC1", Type: genapi.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.CompanyConnectorItemStatusActive, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				},
				Errors: []genapi.ConnectorErrorSummaryItem{
					{Name: "Bad Connector", StatusMessage: "connection timeout"},
				},
				Pagination: genapi.PaginationDTO{HasMore: true, NextCursor: &next},
			}
			page++
		} else if cursor == "comp-page2" {
			resp = genapi.CompanyConnectorListResponse{
				Data: []genapi.CompanyConnectorItem{
					{Id: "cc2", Name: "CC2", Type: genapi.CompanyConnectorItemTypeGoogleReviews, TypeDisplayName: "Google Reviews", Status: genapi.CompanyConnectorItemStatusError, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayError, LocationId: "l2", LocationName: "L2", CreatedAt: now, UpdatedAt: now},
				},
				Errors: []genapi.ConnectorErrorSummaryItem{
					{Name: "Bad Connector", StatusMessage: "connection timeout"},
				},
				Pagination: genapi.PaginationDTO{HasMore: false, NextCursor: nil},
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

		resp := genapi.CompanyConnectorListResponse{
			Data:       []genapi.CompanyConnectorItem{},
			Errors:     []genapi.ConnectorErrorSummaryItem{},
			Pagination: genapi.PaginationDTO{HasMore: false},
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		resp := genapi.CompanyConnectorListResponse{
			Data: []genapi.CompanyConnectorItem{
				{Id: "cc1", Name: "CC1", Type: genapi.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.CompanyConnectorItemStatusActive, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
			},
			Errors:     []genapi.ConnectorErrorSummaryItem{},
			Pagination: genapi.PaginationDTO{HasMore: false},
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
		resp := genapi.LocationConnectorListResponse{
			Data:       []genapi.LocationConnectorItem{},
			Pagination: genapi.PaginationDTO{HasMore: false},
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
		resp := genapi.CompanyConnectorListResponse{
			Data: []genapi.CompanyConnectorItem{
				{Id: "cc1", Name: "CC1", Type: genapi.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.CompanyConnectorItemStatusActive, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				{Id: "cc2", Name: "CC2", Type: genapi.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.CompanyConnectorItemStatusActive, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
				{Id: "cc3", Name: "CC3", Type: genapi.CompanyConnectorItemTypeOoma, TypeDisplayName: "Ooma", Status: genapi.CompanyConnectorItemStatusActive, StatusDisplay: genapi.CompanyConnectorItemStatusDisplayConnected, LocationId: "l1", LocationName: "L1", CreatedAt: now, UpdatedAt: now},
			},
			Errors:     []genapi.ConnectorErrorSummaryItem{},
			Pagination: genapi.PaginationDTO{HasMore: true, NextCursor: &next},
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
