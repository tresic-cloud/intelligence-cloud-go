package intelligencecloud

import (
	"context"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// Type aliases re-export generated reseller types at the package root so
// callers never import internal/generated directly.

// Reseller represents a reseller account in the Intelligence Cloud platform.
type Reseller = generated.Reseller

// CreateResellerRequest is the request body for ResellerService.Create.
type CreateResellerRequest = generated.CreateResellerRequest

// UpdateResellerRequest is the request body for ResellerService.Update
// (full replacement via PUT).
type UpdateResellerRequest = generated.UpdateResellerRequest

// PatchResellerRequest is the request body for ResellerService.Patch
// (partial update via PATCH; all fields optional).
type PatchResellerRequest = generated.PatchResellerRequest

// DeactivateResellerRequest is the request body for
// ResellerService.Deactivate. Reason is required.
type DeactivateResellerRequest = generated.DeactivateResellerRequest

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry = generated.AuditLogEntry

// List returns a paginated iterator over all resellers. Use ListOption
// values such as WithPageSize and WithFilter to control pagination and
// server-side filtering.
func (s *ResellerService) List(ctx context.Context, opts ...ListOption) *Iterator[Reseller] {
	cfg := listConfig{}
	for _, o := range opts {
		o.applyList(&cfg)
	}

	fetcher := func(fetchCtx context.Context, pageToken string) ([]Reseller, string, error) {
		const op = "ListResellers"

		gc, err := s.client.genClient()
		if err != nil {
			return nil, "", err
		}

		params := &generated.ListResellersParams{}
		if pageToken != "" {
			params.Cursor = &pageToken
		}
		if cfg.pageSize > 0 {
			params.Limit = &cfg.pageSize
		}
		if v, ok := cfg.filters["search"]; ok {
			params.Search = &v
		}
		if v, ok := cfg.filters["country"]; ok {
			params.Country = &v
		}
		if v, ok := cfg.filters["is_active"]; ok {
			active := v == "true"
			params.IsActive = &active
		}

		resp, err := gc.ListResellers(fetchCtx, params)
		if err != nil {
			return nil, "", err
		}

		var listResp generated.ResellerListResponse
		if err := handleResponse(resp, op, &listResp); err != nil {
			return nil, "", err
		}

		nextToken := ""
		if listResp.HasMore && listResp.NextCursor != nil {
			nextToken = *listResp.NextCursor
		}
		return listResp.Data, nextToken, nil
	}

	return newIterator(fetcher, cfg.pageSize, cfg.maxItems)
}

// Get retrieves a single reseller by ID.
func (s *ResellerService) Get(ctx context.Context, id string, opts ...CallOption) (*Reseller, error) {
	const op = "GetReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetReseller(ctx, id)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// Create creates a new reseller account.
func (s *ResellerService) Create(ctx context.Context, req CreateResellerRequest, opts ...CallOption) (*Reseller, error) {
	const op = "CreateReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.CreateReseller(ctx, req)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// Update performs a full replacement of a reseller's fields (PUT).
func (s *ResellerService) Update(ctx context.Context, id string, req UpdateResellerRequest, opts ...CallOption) (*Reseller, error) {
	const op = "UpdateReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.UpdateReseller(ctx, id, req)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// Patch performs a partial update of a reseller's fields (PATCH).
func (s *ResellerService) Patch(ctx context.Context, id string, req PatchResellerRequest, opts ...CallOption) (*Reseller, error) {
	const op = "PatchReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.PatchReseller(ctx, id, req)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// Deactivate deactivates a reseller. The reason field in req is required
// for audit purposes.
func (s *ResellerService) Deactivate(ctx context.Context, id string, req DeactivateResellerRequest, opts ...CallOption) (*Reseller, error) {
	const op = "DeactivateReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.DeactivateReseller(ctx, id, req)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// Reactivate reactivates a previously deactivated reseller.
func (s *ResellerService) Reactivate(ctx context.Context, id string, opts ...CallOption) (*Reseller, error) {
	const op = "ReactivateReseller"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.ReactivateReseller(ctx, id)
	if err != nil {
		return nil, err
	}

	var reseller Reseller
	if err := handleResponse(resp, op, &reseller); err != nil {
		return nil, err
	}
	return &reseller, nil
}

// ListAuditLogs returns a paginated iterator over audit log entries for
// the given reseller. Use ListOption values such as WithPageSize and
// WithFilter to control pagination and server-side filtering.
func (s *ResellerService) ListAuditLogs(ctx context.Context, resellerID string, opts ...ListOption) *Iterator[AuditLogEntry] {
	cfg := listConfig{}
	for _, o := range opts {
		o.applyList(&cfg)
	}

	fetcher := func(fetchCtx context.Context, pageToken string) ([]AuditLogEntry, string, error) {
		const op = "ListResellerAuditLogs"

		gc, err := s.client.genClient()
		if err != nil {
			return nil, "", err
		}

		params := &generated.ListResellerAuditLogsParams{}
		if pageToken != "" {
			params.Cursor = &pageToken
		}
		if cfg.pageSize > 0 {
			params.Limit = &cfg.pageSize
		}
		if v, ok := cfg.filters["action"]; ok {
			params.Action = &v
		}
		if v, ok := cfg.filters["start_date"]; ok {
			params.StartDate = &v
		}
		if v, ok := cfg.filters["end_date"]; ok {
			params.EndDate = &v
		}
		if v, ok := cfg.filters["actor_id"]; ok {
			params.ActorId = &v
		}
		if v, ok := cfg.filters["search"]; ok {
			params.Search = &v
		}

		resp, err := gc.ListResellerAuditLogs(fetchCtx, resellerID, params)
		if err != nil {
			return nil, "", err
		}

		var listResp generated.AuditLogListResponse
		if err := handleResponse(resp, op, &listResp); err != nil {
			return nil, "", err
		}

		nextToken := ""
		if listResp.HasMore && listResp.NextCursor != nil {
			nextToken = *listResp.NextCursor
		}
		return listResp.Data, nextToken, nil
	}

	return newIterator(fetcher, cfg.pageSize, cfg.maxItems)
}
