package intelligencecloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// ---------------------------------------------------------------------------
// [B-11] ResellerService.List — paginated
// ---------------------------------------------------------------------------

func TestResellerService_List_SinglePage(t *testing.T) {
	t.Parallel()
	name1 := "Reseller A"
	name2 := "Reseller B"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers" {
			t.Errorf("path = %s, want /api/v1/resellers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data: []generated.Reseller{
				{ResellerName: &name1},
				{ResellerName: &name2},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	var names []string
	for it.Next(context.Background()) {
		r := it.Value()
		if r.ResellerName != nil {
			names = append(names, *r.ResellerName)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d items, want 2", len(names))
	}
	if names[0] != "Reseller A" || names[1] != "Reseller B" {
		t.Errorf("names = %v", names)
	}
}

func TestResellerService_List_MultiPage(t *testing.T) {
	t.Parallel()
	name1 := "Page1"
	name2 := "Page2"
	cursor := "cursor-2"
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			json.NewEncoder(w).Encode(generated.ResellerListResponse{
				Data:       []generated.Reseller{{ResellerName: &name1}},
				HasMore:    true,
				NextCursor: &cursor,
			})
		case 2:
			if r.URL.Query().Get("cursor") != "cursor-2" {
				t.Errorf("cursor = %q, want %q", r.URL.Query().Get("cursor"), "cursor-2")
			}
			json.NewEncoder(w).Encode(generated.ResellerListResponse{
				Data:    []generated.Reseller{{ResellerName: &name2}},
				HasMore: false,
			})
		default:
			t.Error("unexpected page request")
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	var names []string
	for it.Next(context.Background()) {
		r := it.Value()
		if r.ResellerName != nil {
			names = append(names, *r.ResellerName)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d items, want 2", len(names))
	}
	if names[0] != "Page1" || names[1] != "Page2" {
		t.Errorf("names = %v", names)
	}
}

func TestResellerService_List_WithPageSize(t *testing.T) {
	t.Parallel()
	name := "R1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limit := r.URL.Query().Get("limit"); limit != "5" {
			t.Errorf("limit = %q, want %q", limit, "5")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data:    []generated.Reseller{{ResellerName: &name}},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background(), WithPageSize(5))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_List_WithFilter(t *testing.T) {
	t.Parallel()
	name := "US Reseller"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if country := r.URL.Query().Get("country"); country != "US" {
			t.Errorf("country = %q, want %q", country, "US")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.ResellerListResponse{
			Data:    []generated.Reseller{{ResellerName: &name}},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background(), WithFilter("country", "US"))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_List_ErrorMapping(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
		w.Write(errorJSON("FORBIDDEN", "insufficient scopes", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.List(context.Background())
	defer it.Close()

	if it.Next(context.Background()) {
		t.Error("expected Next to return false on error")
	}
	err := it.Err()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthorization) {
		t.Errorf("errors.Is(err, ErrAuthorization) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Get
// ---------------------------------------------------------------------------

func TestResellerService_Get_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Acme Corp"
	active := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
			IsActive:     &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Get(context.Background(), "reseller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.Id == nil || *reseller.Id != "reseller-1" {
		t.Errorf("Id = %v, want reseller-1", reseller.Id)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Acme Corp" {
		t.Errorf("ResellerName = %v, want Acme Corp", reseller.ResellerName)
	}
}

func TestResellerService_Get_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
	var ne *NotFoundError
	if !errors.As(err, &ne) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
}

func TestResellerService_Get_401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

func TestResellerService_Get_500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write(errorJSON("INTERNAL", "server error", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
	if !errors.Is(err, ErrServer) {
		t.Errorf("errors.Is(err, ErrServer) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Create
// ---------------------------------------------------------------------------

func TestResellerService_Create_Success(t *testing.T) {
	t.Parallel()
	id := "new-reseller"
	name := "New Reseller"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers" {
			t.Errorf("path = %s, want /api/v1/resellers", r.URL.Path)
		}

		var body generated.CreateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ResellerName != "New Reseller" {
			t.Errorf("ResellerName = %q, want %q", body.ResellerName, "New Reseller")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Create(context.Background(), CreateResellerRequest{
		ResellerName:        "New Reseller",
		PrimaryContactName:  "Jane Doe",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.Id == nil || *reseller.Id != "new-reseller" {
		t.Errorf("Id = %v, want new-reseller", reseller.Id)
	}
}

func TestResellerService_Create_409(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		w.Write(errorJSON("CONFLICT", "reseller name already exists", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{
		ResellerName:        "Duplicate",
		PrimaryContactName:  "Jane",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("errors.Is(err, ErrConflict) = false; err = %v", err)
	}
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
}

func TestResellerService_Create_422(t *testing.T) {
	t.Parallel()
	field := "reseller_name"
	fieldCode := "required"
	fieldMsg := "reseller_name is required"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if len(ve.FieldErrors) != 1 {
		t.Fatalf("FieldErrors len = %d, want 1", len(ve.FieldErrors))
	}
	if ve.FieldErrors[0].Field != "reseller_name" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "reseller_name")
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Update
// ---------------------------------------------------------------------------

func TestResellerService_Update_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Updated Name"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}

		var body generated.CreateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ResellerName != "Updated Name" {
			t.Errorf("ResellerName = %q, want %q", body.ResellerName, "Updated Name")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Update(context.Background(), "reseller-1", UpdateResellerRequest{
		ResellerName:        "Updated Name",
		PrimaryContactName:  "Jane",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Updated Name" {
		t.Errorf("ResellerName = %v, want Updated Name", reseller.ResellerName)
	}
}

func TestResellerService_Update_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Update(context.Background(), "nonexistent", UpdateResellerRequest{
		ResellerName:        "X",
		PrimaryContactName:  "Y",
		PrimaryContactEmail: "y@example.com",
		PrimaryContactPhone: "+15551234567",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Patch
// ---------------------------------------------------------------------------

func TestResellerService_Patch_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	name := "Patched Name"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req PatchResellerRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.ResellerName == nil || *req.ResellerName != "Patched Name" {
			t.Errorf("ResellerName = %v, want Patched Name", req.ResellerName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:           &id,
			ResellerName: &name,
		})
	}))
	defer srv.Close()

	patchName := "Patched Name"
	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Patch(context.Background(), "reseller-1", PatchResellerRequest{
		ResellerName: &patchName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Patched Name" {
		t.Errorf("ResellerName = %v, want Patched Name", reseller.ResellerName)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Deactivate
// ---------------------------------------------------------------------------

func TestResellerService_Deactivate_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	active := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/deactivate" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/deactivate", r.URL.Path)
		}

		var body generated.DeactivateResellerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Reason != "customer request" {
			t.Errorf("Reason = %q, want %q", body.Reason, "customer request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:       &id,
			IsActive: &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Deactivate(context.Background(), "reseller-1", DeactivateResellerRequest{
		Reason: "customer request",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.IsActive == nil || *reseller.IsActive != false {
		t.Errorf("IsActive = %v, want false", reseller.IsActive)
	}
}

func TestResellerService_Deactivate_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Deactivate(context.Background(), "nonexistent", DeactivateResellerRequest{
		Reason: "test",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

func TestResellerService_Deactivate_422(t *testing.T) {
	t.Parallel()
	field := "reason"
	fieldCode := "required"
	fieldMsg := "reason is required"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		w.Write(errorJSON("VALIDATION_ERROR", "validation failed", []generated.ErrorDetail{
			{Field: &field, Code: &fieldCode, Message: &fieldMsg},
		}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Deactivate(context.Background(), "reseller-1", DeactivateResellerRequest{})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if len(ve.FieldErrors) != 1 {
		t.Fatalf("FieldErrors len = %d, want 1", len(ve.FieldErrors))
	}
	if ve.FieldErrors[0].Field != "reason" {
		t.Errorf("FieldErrors[0].Field = %q, want %q", ve.FieldErrors[0].Field, "reason")
	}
}

// ---------------------------------------------------------------------------
// ResellerService.Reactivate
// ---------------------------------------------------------------------------

func TestResellerService_Reactivate_Success(t *testing.T) {
	t.Parallel()
	id := "reseller-1"
	active := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/reactivate" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/reactivate", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.Reseller{
			Id:       &id,
			IsActive: &active,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	reseller, err := c.Resellers.Reactivate(context.Background(), "reseller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reseller.IsActive == nil || !*reseller.IsActive {
		t.Errorf("IsActive = %v, want true", reseller.IsActive)
	}
}

func TestResellerService_Reactivate_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		w.Write(errorJSON("NOT_FOUND", "reseller not found", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Reactivate(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService.ListAuditLogs
// ---------------------------------------------------------------------------

func TestResellerService_ListAuditLogs_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/resellers/reseller-1/audit-logs" {
			t.Errorf("path = %s, want /api/v1/resellers/reseller-1/audit-logs", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AuditLogListResponse{
			Data: []generated.AuditLogEntry{
				{
					Id:        "log-1",
					Action:    "reseller.updated",
					ActorId:   "user-1",
					ActorName: "Admin",
					Details:   "Updated reseller name",
					Timestamp: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1")
	defer it.Close()

	var entries []AuditLogEntry
	for it.Next(context.Background()) {
		entries = append(entries, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Action != "reseller.updated" {
		t.Errorf("Action = %q, want %q", entries[0].Action, "reseller.updated")
	}
}

func TestResellerService_ListAuditLogs_WithFilter(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if action := r.URL.Query().Get("action"); action != "reseller.deactivated" {
			t.Errorf("action = %q, want %q", action, "reseller.deactivated")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generated.AuditLogListResponse{
			Data:    []generated.AuditLogEntry{},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1",
		WithFilter("action", "reseller.deactivated"))
	defer it.Close()

	for it.Next(context.Background()) {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResellerService_ListAuditLogs_ErrorMapping(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write(errorJSON("UNAUTHENTICATED", "invalid token", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	it := c.Resellers.ListAuditLogs(context.Background(), "reseller-1")
	defer it.Close()

	if it.Next(context.Background()) {
		t.Error("expected Next to return false on error")
	}
	err := it.Err()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("errors.Is(err, ErrAuthentication) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResellerService — 429 Rate Limit
// ---------------------------------------------------------------------------

func TestResellerService_Get_429(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(429)
		w.Write(errorJSON("RATE_LIMITED", "too many requests", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Get(context.Background(), "id")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRateLimit) {
		t.Errorf("errors.Is(err, ErrRateLimit) = false; err = %v", err)
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("errors.As failed; err = %T: %v", err, err)
	}
	if rl.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want 60s", rl.RetryAfter)
	}
}

// ---------------------------------------------------------------------------
// ResellerService — 400 Bad Request
// ---------------------------------------------------------------------------

func TestResellerService_Create_400(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		w.Write(errorJSON("BAD_REQUEST", "malformed request body", nil))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resellers.Create(context.Background(), CreateResellerRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", err)
	}
}
