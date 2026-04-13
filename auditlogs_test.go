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
		resp := genapi.AuditLogListResponse{
			Data: []genapi.AuditLogEntry{
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

		var resp genapi.AuditLogListResponse
		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

		if page == 0 && cursor == "" {
			next := "cursor-page2"
			resp = genapi.AuditLogListResponse{
				Data: []genapi.AuditLogEntry{
					{Id: "e1", Action: "a1", ActorId: "u1", ActorName: "U1", Details: "d1", Timestamp: now},
				},
				HasMore:    true,
				NextCursor: &next,
			}
			page++
		} else if cursor == "cursor-page2" {
			resp = genapi.AuditLogListResponse{
				Data: []genapi.AuditLogEntry{
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

		resp := genapi.AuditLogListResponse{
			Data:    []genapi.AuditLogEntry{},
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		resp := genapi.AuditLogListResponse{
			Data:    []genapi.AuditLogEntry{},
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		json.NewEncoder(w).Encode(genapi.ErrorResponse{
			Error: genapi.Error{
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
		resp := genapi.AuditLogListResponse{
			Data: []genapi.AuditLogEntry{
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
