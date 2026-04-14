package intelligencecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// ─── Error Mapping ───────────────────────────────────────────────

// maxErrBodyBytes caps the amount of response body read for error mapping.
const maxErrBodyBytes = 4096

// apiErrEnvelope matches the generated ErrorResponse shape: { "error": { ... } }.
type apiErrEnvelope struct {
	Error struct {
		Code      string                  `json:"code"`
		Message   string                  `json:"message"`
		Details   []generated.ErrorDetail `json:"details"`
		RequestID *string                 `json:"request_id"`
	} `json:"error"`
}

// mapResponseError reads a non-2xx HTTP response body and returns the
// appropriate typed SDK error. It is used by all resource-service wrappers.
// The caller must NOT have already consumed resp.Body.
func mapResponseError(resp *http.Response, operationID string) error {
	if resp == nil {
		return fmt.Errorf("intelligencecloud: nil response for operation %s", operationID)
	}

	var raw []byte
	if resp.Body != nil {
		raw, _ = io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
		resp.Body.Close()
	}

	var env apiErrEnvelope
	_ = json.Unmarshal(raw, &env)

	code := env.Error.Code
	message := env.Error.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	requestID := ""
	if env.Error.RequestID != nil {
		requestID = *env.Error.RequestID
	}
	if rid := resp.Header.Get("X-Request-Id"); rid != "" {
		requestID = rid
	}

	switch {
	case resp.StatusCode == 400 || resp.StatusCode == 422:
		var fieldErrors []FieldError
		for _, d := range env.Error.Details {
			fe := FieldError{}
			if d.Field != nil {
				fe.Field = *d.Field
			}
			if d.Code != nil {
				fe.Code = *d.Code
			}
			if d.Message != nil {
				fe.Message = *d.Message
			}
			fieldErrors = append(fieldErrors, fe)
		}
		return NewValidationError(resp.StatusCode, code, message, requestID, operationID, fieldErrors)

	case resp.StatusCode == 401:
		return NewAuthenticationError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 403:
		return NewAuthorizationError(resp.StatusCode, code, message, requestID, operationID, nil)

	case resp.StatusCode == 404:
		return NewNotFoundError(resp.StatusCode, code, message, requestID, operationID, "", "")

	case resp.StatusCode == 409:
		return NewConflictError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 429:
		retryAfter := time.Duration(0)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				retryAfter = time.Duration(secs) * time.Second
			}
		}
		return NewRateLimitError(resp.StatusCode, code, message, requestID, operationID, retryAfter)

	case resp.StatusCode >= 500:
		return NewServerError(resp.StatusCode, code, message, requestID, operationID)

	default:
		return NewUnexpectedError(resp.StatusCode, code, message, requestID, operationID, raw)
	}
}

// genClient creates a generated.Client wired to the SDK Client's base URL
// and HTTP client. The generated client is lightweight and safe to create
// per-call; it holds only a server string and *http.Client reference.
func (c *Client) genClient() (*generated.Client, error) {
	return generated.NewClient(
		c.baseURL.String(),
		generated.WithHTTPClient(c.httpClient),
	)
}

// decodeResponse reads and JSON-decodes the response body into v, then
// closes the body. It returns an error if decoding fails.
func decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// handleResponse checks the HTTP status code and either decodes a
// successful response or maps the error. For responses with a status code
// in [200,299] the body is decoded into v. Otherwise a typed SDK error is
// returned.
func handleResponse(resp *http.Response, operationID string, v interface{}) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if v == nil {
			// No response body expected (e.g. 204 No Content).
			if resp.Body != nil {
				resp.Body.Close()
			}
			return nil
		}
		return decodeResponse(resp, v)
	}
	return mapResponseError(resp, operationID)
}

// mapStandardError maps a non-2xx response with the standard ErrorResponse
// envelope to a typed SDK error. This helper is used by list operations and
// other endpoints that use the standard error shape.
func mapStandardError(statusCode int, body []byte, requestID, operationID string) error {
	code, message := parseStandardErrorBody(body)

	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch {
	case statusCode == 400 || statusCode == 422:
		return NewValidationError(statusCode, code, message, requestID, operationID, nil)
	case statusCode == 401:
		return NewAuthenticationError(statusCode, code, message, requestID, operationID)
	case statusCode == 403:
		return NewAuthorizationError(statusCode, code, message, requestID, operationID, nil)
	case statusCode == 404:
		return NewNotFoundError(statusCode, code, message, requestID, operationID, "", "")
	case statusCode == 409:
		return NewConflictError(statusCode, code, message, requestID, operationID)
	case statusCode == 429:
		return NewRateLimitError(statusCode, code, message, requestID, operationID, time.Duration(0))
	case statusCode >= 500:
		return NewServerError(statusCode, code, message, requestID, operationID)
	default:
		return NewUnexpectedError(statusCode, code, message, requestID, operationID, body)
	}
}

// parseStandardErrorBody parses the standard ErrorResponse envelope, returning
// the error code and message.
func parseStandardErrorBody(body []byte) (code, message string) {
	if len(body) == 0 {
		return "", ""
	}

	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Code != "" {
		return envelope.Error.Code, envelope.Error.Message
	}

	return "", ""
}

// resolveCallOpts applies all CallOption values and returns the resolved config.
func resolveCallOpts(opts []CallOption) callConfig {
	var cfg callConfig
	for _, o := range opts {
		o.applyCall(&cfg)
	}
	return cfg
}

// applyCallHeaders sets extra headers from the resolved call config on req.
func applyCallHeaders(req *http.Request, cfg callConfig) {
	if cfg.idempotencyKey != "" {
		req.Header.Set("X-Idempotency-Key", cfg.idempotencyKey)
	}
	for k, v := range cfg.extraHeaders {
		req.Header.Set(k, v)
	}
}

// do executes an HTTP request with auth injection and error mapping.
// On success (2xx/3xx), it returns the *http.Response. On error
// (4xx/5xx or transport failure), it returns a typed error.
func (c *Client) do(ctx context.Context, req *http.Request, operationID string) (*http.Response, error) {
	tok, err := c.credentialProvider.Token(ctx)
	if err != nil {
		return nil, NewAuthenticationError(0, "", err.Error(), "", operationID)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return resp, nil
	}

	defer resp.Body.Close()
	return nil, mapHTTPError(resp, operationID)
}

// mapHTTPError reads an error response body and maps the HTTP status to
// the appropriate typed SDK error.
func mapHTTPError(resp *http.Response, operationID string) error {
	requestID := resp.Header.Get("X-Request-Id")

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var parsed struct {
		Code           string       `json:"code"`
		Message        string       `json:"message"`
		FieldErrors    []FieldError `json:"field_errors"`
		RequiredScopes []string     `json:"required_scopes"`
		ResourceType   string       `json:"resource_type"`
		ResourceID     string       `json:"resource_id"`
	}
	_ = json.Unmarshal(body, &parsed)

	code := parsed.Code
	message := parsed.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	switch {
	case resp.StatusCode == 400 || resp.StatusCode == 422:
		return NewValidationError(resp.StatusCode, code, message, requestID, operationID, parsed.FieldErrors)
	case resp.StatusCode == 401:
		return NewAuthenticationError(resp.StatusCode, code, message, requestID, operationID)
	case resp.StatusCode == 403:
		return NewAuthorizationError(resp.StatusCode, code, message, requestID, operationID, parsed.RequiredScopes)
	case resp.StatusCode == 404:
		return NewNotFoundError(resp.StatusCode, code, message, requestID, operationID, parsed.ResourceType, parsed.ResourceID)
	case resp.StatusCode == 409:
		return NewConflictError(resp.StatusCode, code, message, requestID, operationID)
	case resp.StatusCode >= 500:
		return NewServerError(resp.StatusCode, code, message, requestID, operationID)
	default:
		return NewUnexpectedError(resp.StatusCode, code, message, requestID, operationID, body)
	}
}

// ─── MeService ───────────────────────────────────────────────────

// Type aliases re-export generated types at the package root so callers
// never import internal/generated directly.

// UserProfile is the authenticated user's profile returned by MeService.Get.
type UserProfile = generated.UserProfile

// AvatarUploadURLRequest is the request body for MeService.CreateAvatarUploadURL.
type AvatarUploadURLRequest = generated.AvatarUploadURLRequest

// AvatarUploadURLResponseData is the response from MeService.CreateAvatarUploadURL.
type AvatarUploadURLResponseData = generated.AvatarUploadURLResponseData

// AvatarConfirmRequest is the request body for MeService.ConfirmAvatar.
type AvatarConfirmRequest = generated.AvatarConfirmRequest

// AvatarResponseData is the response from MeService.ConfirmAvatar.
type AvatarResponseData = generated.AvatarResponseData

// NotificationPreferences holds the user's notification preference settings.
type NotificationPreferences = generated.NotificationPreferences

// DailyRecapPreferences holds the user's daily recap preference settings.
type DailyRecapPreferences = generated.DailyRecapPreferences

// NotificationPreferencesPatch is the request body for
// MeService.PatchNotificationPreferences (all fields optional).
type NotificationPreferencesPatch = generated.PatchNotificationPreferencesRequest

// DailyRecapPreferencesPatch is the request body for
// MeService.PatchDailyRecapPreferences (all fields optional).
type DailyRecapPreferencesPatch = generated.PatchDailyRecapPreferencesRequest

// Get retrieves the authenticated user's profile.
func (s *MeService) Get(ctx context.Context, opts ...CallOption) (*UserProfile, error) {
	const op = "GetMe"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetMe(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.UserProfileResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// CreateAvatarUploadURL generates a pre-signed upload URL for a profile
// photo. After uploading the image to the returned URL, call ConfirmAvatar
// to store the asset on the profile.
func (s *MeService) CreateAvatarUploadURL(ctx context.Context, req AvatarUploadURLRequest, opts ...CallOption) (*AvatarUploadURLResponseData, error) {
	const op = "CreateAvatarUploadURL"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.CreateAvatarUploadURL(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.AvatarUploadURLResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// ConfirmAvatar confirms a previously generated avatar upload by storing
// the asset URL on the user profile.
func (s *MeService) ConfirmAvatar(ctx context.Context, req AvatarConfirmRequest, opts ...CallOption) (*AvatarResponseData, error) {
	const op = "ConfirmAvatar"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.ConfirmAvatar(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.AvatarResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// DeleteAvatar removes the authenticated user's profile photo.
func (s *MeService) DeleteAvatar(ctx context.Context, opts ...CallOption) error {
	const op = "DeleteAvatar"

	gc, err := s.client.genClient()
	if err != nil {
		return err
	}

	resp, err := gc.DeleteAvatar(ctx)
	if err != nil {
		return err
	}

	return handleResponse(resp, op, nil)
}

// GetNotificationPreferences retrieves the authenticated user's
// notification preference settings.
func (s *MeService) GetNotificationPreferences(ctx context.Context, opts ...CallOption) (*NotificationPreferences, error) {
	const op = "GetNotificationPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetNotificationPreferences(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.NotificationPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// PatchNotificationPreferences partially updates the authenticated user's
// notification preferences. Only the fields present in req are modified.
func (s *MeService) PatchNotificationPreferences(ctx context.Context, req NotificationPreferencesPatch, opts ...CallOption) (*NotificationPreferences, error) {
	const op = "PatchNotificationPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.PatchNotificationPreferences(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.NotificationPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// GetDailyRecapPreferences retrieves the authenticated user's daily recap
// preference settings.
func (s *MeService) GetDailyRecapPreferences(ctx context.Context, opts ...CallOption) (*DailyRecapPreferences, error) {
	const op = "GetDailyRecapPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetDailyRecapPreferences(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.DailyRecapPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// PatchDailyRecapPreferences partially updates the authenticated user's
// daily recap preferences. Only the fields present in req are modified.
func (s *MeService) PatchDailyRecapPreferences(ctx context.Context, req DailyRecapPreferencesPatch, opts ...CallOption) (*DailyRecapPreferences, error) {
	const op = "PatchDailyRecapPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.PatchDailyRecapPreferences(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.DailyRecapPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// ─── ResellerService ─────────────────────────────────────────────

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

// AuditLogEntry type alias is declared in the AuditLogService section below.

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

// ─── AuthService ─────────────────────────────────────────────────

// LoginRequest contains the credentials for authenticating with the
// Intelligence Cloud API.
type LoginRequest = generated.LoginRequest

// LoginResponse contains the tokens and user profile returned on
// successful authentication.
type LoginResponse = generated.LoginResponse

// Login authenticates with the Intelligence Cloud API using email and password
// credentials. This endpoint uses security: [] in the OpenAPI spec, so NO
// Authorization header is sent. The method handles both OAuth-style error
// envelopes (error + error_description) and standard ErrorResponse envelopes.
func (s *AuthService) Login(ctx context.Context, req LoginRequest, opts ...CallOption) (*LoginResponse, error) {
	const operationID = "Login"

	// Apply call options.
	var cfg callConfig
	for _, opt := range opts {
		opt.applyCall(&cfg)
	}

	// Apply per-call timeout if configured.
	if cfg.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.requestTimeout)
		defer cancel()
	}

	// Build the request using the generated request builder.
	httpReq, err := generated.NewLoginRequest(s.client.baseURL.String(), req)
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: failed to build login request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	// Apply extra headers (but never Authorization — this endpoint bypasses auth).
	for k, v := range cfg.extraHeaders {
		httpReq.Header.Set(k, v)
	}

	// Execute the request directly via the HTTP client, bypassing the
	// transport's auth injection. Login uses security: [] in the OpenAPI spec.
	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body (capped to prevent unbounded reads).
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: failed to read login response: %w", err)
	}

	// Capture X-Request-Id if present.
	requestID := resp.Header.Get("X-Request-Id")

	// Handle success.
	if resp.StatusCode == http.StatusOK {
		var loginResp LoginResponse
		if err := json.Unmarshal(body, &loginResp); err != nil {
			return nil, fmt.Errorf("intelligencecloud: failed to decode login response: %w", err)
		}
		return &loginResp, nil
	}

	// Handle error — try OAuth envelope first, fall back to standard.
	return nil, mapLoginError(resp.StatusCode, body, requestID, operationID)
}

// oauthErrorBody is the OAuth-style error envelope used by /api/v1/auth/login.
type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// standardErrorBody is the standard error envelope used by most endpoints.
type standardErrorBody struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

// mapLoginError maps a non-2xx login response to a typed SDK error. It first
// tries parsing as an OAuth error envelope, then falls back to the standard
// ErrorResponse envelope.
func mapLoginError(statusCode int, body []byte, requestID, operationID string) error {
	code, message := parseLoginErrorBody(body)

	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch {
	case statusCode == 400 || statusCode == 422:
		return NewValidationError(statusCode, code, message, requestID, operationID, nil)
	case statusCode == 401:
		return NewAuthenticationError(statusCode, code, message, requestID, operationID)
	case statusCode == 403:
		return NewAuthorizationError(statusCode, code, message, requestID, operationID, nil)
	case statusCode == 404:
		return NewNotFoundError(statusCode, code, message, requestID, operationID, "", "")
	case statusCode == 409:
		return NewConflictError(statusCode, code, message, requestID, operationID)
	case statusCode == 429:
		return NewRateLimitError(statusCode, code, message, requestID, operationID, time.Duration(0))
	case statusCode >= 500:
		return NewServerError(statusCode, code, message, requestID, operationID)
	default:
		return NewUnexpectedError(statusCode, code, message, requestID, operationID, body)
	}
}

// parseLoginErrorBody tries to parse the response body as an OAuth error
// envelope first, then as a standard error envelope. Returns the error code
// and message.
func parseLoginErrorBody(body []byte) (code, message string) {
	if len(body) == 0 {
		return "", ""
	}

	// Try OAuth envelope first.
	var oauth oauthErrorBody
	if err := json.Unmarshal(body, &oauth); err == nil && oauth.Error != "" {
		return oauth.Error, oauth.ErrorDescription
	}

	// Fall back to standard error envelope.
	var std standardErrorBody
	if err := json.Unmarshal(body, &std); err == nil && std.Error.Code != "" {
		return std.Error.Code, std.Error.Message
	}

	return "", ""
}

// ─── AuditLogService ─────────────────────────────────────────────

// AuditLogEntry is a single audit log entry from the Intelligence Cloud API.
type AuditLogEntry = generated.AuditLogEntry

// ListForReseller returns a paginated iterator over audit log entries for the
// specified reseller. Filters can be applied using WithFilter with keys:
// "action", "start_date", "end_date", "actor_id", "search". Use WithPageSize
// to control page size, and WithMaxItems to cap total results.
func (s *AuditLogService) ListForReseller(ctx context.Context, resellerID string, opts ...ListOption) *Iterator[AuditLogEntry] {
	var cfg listConfig
	for _, opt := range opts {
		opt.applyList(&cfg)
	}

	fetch := func(fetchCtx context.Context, pageToken string) ([]AuditLogEntry, string, error) {
		params := &generated.ListResellerAuditLogsParams{}

		// Wire cursor for pagination.
		if pageToken != "" {
			params.Cursor = &pageToken
		}

		// Wire page size.
		if cfg.pageSize > 0 {
			params.Limit = &cfg.pageSize
		}

		// Wire filters.
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

		// Build the request using the generated request builder.
		httpReq, err := generated.NewListResellerAuditLogsRequest(s.client.baseURL.String(), resellerID, params)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to build audit log list request: %w", err)
		}
		httpReq = httpReq.WithContext(fetchCtx)

		// Execute the request.
		resp, err := s.client.httpClient.Do(httpReq)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: audit log list request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to read audit log list response: %w", err)
		}

		requestID := resp.Header.Get("X-Request-Id")

		if resp.StatusCode != http.StatusOK {
			return nil, "", mapStandardError(resp.StatusCode, body, requestID, "ListResellerAuditLogs")
		}

		var listResp generated.AuditLogListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to decode audit log list response: %w", err)
		}

		nextToken := ""
		if listResp.HasMore && listResp.NextCursor != nil {
			nextToken = *listResp.NextCursor
		}

		return listResp.Data, nextToken, nil
	}

	return newIterator(fetch, cfg.pageSize, cfg.maxItems)
}

// ─── ConnectorService ────────────────────────────────────────────

// LocationConnector is a connector scoped to a single location.
type LocationConnector = generated.LocationConnectorItem

// CompanyConnector is a connector scoped to a company, including its location
// association.
type CompanyConnector = generated.CompanyConnectorItem

// ConnectorErrorSummaryItem is an aggregate error summary for connectors in
// error state across a company.
type ConnectorErrorSummaryItem = generated.ConnectorErrorSummaryItem

// CompanyConnectorList holds both the paginated connector iterator and the
// aggregate error summary returned by the company-scoped connectors endpoint.
// The ErrorSummary reflects all connectors in error state across the entire
// company, regardless of pagination.
type CompanyConnectorList struct {
	// Iterator provides paginated access to company connectors.
	Iterator *Iterator[CompanyConnector]

	// ErrorSummary lists all connectors in error state across the company.
	// This is populated from the first page response and represents the
	// complete set, independent of pagination.
	ErrorSummary []ConnectorErrorSummaryItem
}

// ListByLocation returns a paginated iterator over connectors for the
// specified location. Filters can be applied using WithFilter with keys:
// "search", "status". Use WithPageSize to control page size.
func (s *ConnectorService) ListByLocation(ctx context.Context, locationID string, opts ...ListOption) *Iterator[LocationConnector] {
	var cfg listConfig
	for _, opt := range opts {
		opt.applyList(&cfg)
	}

	fetch := func(fetchCtx context.Context, pageToken string) ([]LocationConnector, string, error) {
		params := &generated.ListLocationConnectorsParams{}

		if pageToken != "" {
			params.Cursor = &pageToken
		}
		if cfg.pageSize > 0 {
			params.Limit = &cfg.pageSize
		}
		if v, ok := cfg.filters["search"]; ok {
			params.Search = &v
		}
		if v, ok := cfg.filters["status"]; ok {
			params.Status = &v
		}

		httpReq, err := generated.NewListLocationConnectorsRequest(s.client.baseURL.String(), locationID, params)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to build location connector list request: %w", err)
		}
		httpReq = httpReq.WithContext(fetchCtx)

		resp, err := s.client.httpClient.Do(httpReq)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: location connector list request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to read location connector list response: %w", err)
		}

		requestID := resp.Header.Get("X-Request-Id")

		if resp.StatusCode != http.StatusOK {
			return nil, "", mapStandardError(resp.StatusCode, body, requestID, "ListLocationConnectors")
		}

		var listResp generated.LocationConnectorListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to decode location connector list response: %w", err)
		}

		nextToken := ""
		if listResp.Pagination.HasMore && listResp.Pagination.NextCursor != nil {
			nextToken = *listResp.Pagination.NextCursor
		}

		return listResp.Data, nextToken, nil
	}

	return newIterator(fetch, cfg.pageSize, cfg.maxItems)
}

// ListByCompany returns a CompanyConnectorList containing both a paginated
// iterator over company-scoped connectors and an aggregate error summary.
// The company connectors endpoint uses nested pagination
// (pagination.next_cursor + pagination.has_more) and carries a top-level
// errors array with connector-scoped aggregate diagnostics.
//
// The first page is fetched eagerly so that ErrorSummary is available
// immediately without calling Iterator.Next(). If the first page fetch fails,
// ErrorSummary will be empty and the error will surface when Iterator.Next()
// is called.
//
// Filters can be applied using WithFilter with keys: "search", "status",
// "type". Use WithPageSize to control page size.
func (s *ConnectorService) ListByCompany(ctx context.Context, companyID string, opts ...ListOption) *CompanyConnectorList {
	var cfg listConfig
	for _, opt := range opts {
		opt.applyList(&cfg)
	}

	result := &CompanyConnectorList{}

	// Eagerly fetch the first page to populate ErrorSummary.
	firstPage, firstNextToken, firstErr := fetchCompanyConnectorPage(
		ctx, s.client, companyID, "", &cfg,
	)

	if firstErr == nil {
		result.ErrorSummary = firstPage.errors
	}

	// Build a fetcher that returns the cached first page on the first call,
	// then fetches subsequent pages from the API.
	firstPageConsumed := false
	fetch := func(fetchCtx context.Context, pageToken string) ([]CompanyConnector, string, error) {
		if !firstPageConsumed {
			firstPageConsumed = true
			if firstErr != nil {
				return nil, "", firstErr
			}
			return firstPage.data, firstNextToken, nil
		}

		page, nextToken, err := fetchCompanyConnectorPage(
			fetchCtx, s.client, companyID, pageToken, &cfg,
		)
		if err != nil {
			return nil, "", err
		}
		return page.data, nextToken, nil
	}

	result.Iterator = newIterator(fetch, cfg.pageSize, cfg.maxItems)

	return result
}

// companyConnectorPage holds the parsed response from one page of company
// connectors, including the error summary.
type companyConnectorPage struct {
	data   []CompanyConnector
	errors []ConnectorErrorSummaryItem
}

// fetchCompanyConnectorPage fetches a single page of company connectors.
func fetchCompanyConnectorPage(
	ctx context.Context,
	c *Client,
	companyID string,
	pageToken string,
	cfg *listConfig,
) (companyConnectorPage, string, error) {
	params := &generated.ListCompanyConnectorsParams{}

	if pageToken != "" {
		params.Cursor = &pageToken
	}
	if cfg.pageSize > 0 {
		params.Limit = &cfg.pageSize
	}
	if v, ok := cfg.filters["search"]; ok {
		params.Search = &v
	}
	if v, ok := cfg.filters["status"]; ok {
		params.Status = &v
	}
	if v, ok := cfg.filters["type"]; ok {
		params.Type = &v
	}

	httpReq, err := generated.NewListCompanyConnectorsRequest(c.baseURL.String(), companyID, params)
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to build company connector list request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: company connector list request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to read company connector list response: %w", err)
	}

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != http.StatusOK {
		return companyConnectorPage{}, "", mapStandardError(resp.StatusCode, body, requestID, "ListCompanyConnectors")
	}

	var listResp generated.CompanyConnectorListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to decode company connector list response: %w", err)
	}

	nextToken := ""
	if listResp.Pagination.HasMore && listResp.Pagination.NextCursor != nil {
		nextToken = *listResp.Pagination.NextCursor
	}

	return companyConnectorPage{
		data:   listResp.Data,
		errors: listResp.Errors,
	}, nextToken, nil
}

// ─── ProductService ──────────────────────────────────────────────

// Product represents a product in the Intelligence Cloud catalogue.
type Product = generated.Product

// List retrieves all products from the Intelligence Cloud API. The
// /api/v1/products endpoint returns all products in a single response
// (no pagination). Error mapping covers 401 (AuthenticationError) and
// 403 (AuthorizationError).
func (s *ProductService) List(ctx context.Context, opts ...CallOption) ([]Product, error) {
	cfg := resolveCallOpts(opts)

	if cfg.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.requestTimeout)
		defer cancel()
	}

	req, err := generated.NewListProductsRequest(s.client.baseURL.String())
	if err != nil {
		return nil, fmt.Errorf("intelligencecloud: build request: %w", err)
	}
	req = req.WithContext(ctx)

	applyCallHeaders(req, cfg)

	resp, err := s.client.do(ctx, req, "ListProducts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var envelope generated.ProductListResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("intelligencecloud: decode response: %w", err)
	}

	if envelope.Data == nil {
		return []Product{}, nil
	}
	return envelope.Data, nil
}

// ─── UserService ─────────────────────────────────────────────────

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
