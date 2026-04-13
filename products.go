// Package intelligencecloud -- ProductService operations.
//
// ProductService exposes product-catalogue endpoints from the Intelligence
// Cloud API. The ProductService type itself is declared alongside the
// top-level Client in client.go.
package intelligencecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

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

// ---------------------------------------------------------------------------
// Shared per-file helpers — kept small; duplication is acceptable across
// service files until a shared errmap.go is established.
// ---------------------------------------------------------------------------

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
