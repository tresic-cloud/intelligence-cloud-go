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

// AuditLogEntry is a single audit log entry from the Intelligence Cloud API.
type AuditLogEntry = genapi.AuditLogEntry

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
		params := &genapi.ListResellerAuditLogsParams{}

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
		httpReq, err := genapi.NewListResellerAuditLogsRequest(s.client.baseURL.String(), resellerID, params)
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

		var listResp genapi.AuditLogListResponse
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
