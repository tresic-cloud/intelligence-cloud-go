// Package intelligencecloud -- UserService operations.
//
// UserService exposes user-management endpoints from the Intelligence
// Cloud API. The UserService type itself is declared alongside the
// top-level Client in client.go.
package intelligencecloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// CompanyUser represents a user within a company.
type CompanyUser = generated.CompanyUserResponseData

// UpdateCompanyUserRequest is the request body for patching a company user.
type UpdateCompanyUserRequest = generated.UpdateCompanyUserRequest

// PatchCompany updates a user within a company. It sends a PATCH request
// to /api/v1/companies/{companyId}/users/{userId}. Error mapping covers
// 400 (ValidationError), 401 (AuthenticationError), 403
// (AuthorizationError), and 404 (NotFoundError).
//
// Both companyID and userID must be non-empty; a ValidationError is
// returned if either is blank.
func (s *UserService) PatchCompany(ctx context.Context, companyID, userID string, req UpdateCompanyUserRequest, opts ...CallOption) (*CompanyUser, error) {
	const op = "PatchCompanyUser"

	if companyID == "" {
		return nil, NewValidationError(0, "invalid_argument", "companyID must not be empty", "", op, nil)
	}
	if userID == "" {
		return nil, NewValidationError(0, "invalid_argument", "userID must not be empty", "", op, nil)
	}

	cfg := resolveCallOpts(opts)

	if cfg.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.requestTimeout)
		defer cancel()
	}

	httpReq, err := generated.NewPatchCompanyUserRequest(
		s.client.baseURL.String(),
		companyID,
		userID,
		generated.PatchCompanyUserJSONRequestBody(req),
	)
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: build request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	applyCallHeaders(httpReq, cfg)

	resp, err := s.client.do(ctx, httpReq, op)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var envelope generated.CompanyUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("intelligencecloud: decode response: %w", err)
	}

	return &envelope.Data, nil
}
