package intelligencecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	genapi "github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// LoginRequest contains the credentials for authenticating with the
// Intelligence Cloud API.
type LoginRequest = genapi.LoginRequest

// LoginResponse contains the tokens and user profile returned on
// successful authentication.
type LoginResponse = genapi.LoginResponse

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
	httpReq, err := genapi.NewLoginRequest(s.client.baseURL.String(), req)
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
