// Package intelligencecloud provides a typed Go client for the Intelligence
// Cloud API. It exposes resource services, typed errors, retry policy, and
// per-call/per-list options.
package intelligencecloud

import (
	"errors"
	"fmt"
	"time"
)

// maxCauseBodyBytes is the hard cap on UnexpectedError.CauseBody to prevent
// unbounded memory from large error responses.
const maxCauseBodyBytes = 4096

// Sentinel errors allow callers to classify failures with errors.Is without
// inspecting concrete types. Every concrete error Unwraps to the
// corresponding sentinel.
var (
	// ErrAuthentication indicates a 401 authentication failure.
	ErrAuthentication = errors.New("intelligencecloud: authentication failed")
	// ErrAuthorization indicates a 403 authorization failure.
	ErrAuthorization = errors.New("intelligencecloud: not authorized")
	// ErrValidation indicates a 400/422 validation failure.
	ErrValidation = errors.New("intelligencecloud: validation failed")
	// ErrNotFound indicates a 404 resource-not-found response.
	ErrNotFound = errors.New("intelligencecloud: resource not found")
	// ErrConflict indicates a 409 resource conflict.
	ErrConflict = errors.New("intelligencecloud: resource conflict")
	// ErrRateLimit indicates a 429 rate-limit response.
	ErrRateLimit = errors.New("intelligencecloud: rate-limited")
	// ErrServer indicates a 5xx server-side failure.
	ErrServer = errors.New("intelligencecloud: server error")
)

// APIError is satisfied by every non-nil error the SDK returns from an API
// call. Callers can use errors.As to obtain the concrete type and access
// additional fields such as RequiredScopes, FieldErrors, or RetryAfter.
type APIError interface {
	error
	// Status returns the HTTP status code of the failed response.
	Status() int
	// Code returns the backend-provided error code, if any.
	Code() string
	// RequestID returns the backend-provided request identifier for support
	// triage.
	RequestID() string
	// Operation returns the SDK operation name that triggered the error.
	Operation() string
}

// FieldError describes a single field-level validation failure as reported by
// the backend.
type FieldError struct {
	// Field is the path of the invalid field (e.g. "name" or "address.zip").
	Field string
	// Code is the machine-readable validation code (e.g. "required").
	Code string
	// Message is the human-readable description of the failure.
	Message string
}

// ---------------------------------------------------------------------------
// Concrete error types
// ---------------------------------------------------------------------------

// apiErrorBase holds the common fields shared by all concrete API error types.
// It is unexported to keep the public surface area clean while satisfying the
// APIError interface methods.
type apiErrorBase struct {
	statusCode  int
	errorCode   string
	message     string
	requestID   string
	operationID string
}

// Status returns the HTTP status code.
func (b apiErrorBase) Status() int { return b.statusCode }

// Code returns the backend error code.
func (b apiErrorBase) Code() string { return b.errorCode }

// RequestID returns the request identifier.
func (b apiErrorBase) RequestID() string { return b.requestID }

// Operation returns the operation name.
func (b apiErrorBase) Operation() string { return b.operationID }

// AuthenticationError represents a 401 authentication failure.
type AuthenticationError struct {
	apiErrorBase
}

// NewAuthenticationError constructs an AuthenticationError.
func NewAuthenticationError(statusCode int, code, message, requestID, operationID string) *AuthenticationError {
	return &AuthenticationError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
	}
}

// Error implements the error interface.
func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s)",
		ErrAuthentication.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *AuthenticationError) Unwrap() error { return ErrAuthentication }

// AuthorizationError represents a 403 authorization failure.
type AuthorizationError struct {
	apiErrorBase
	// RequiredScopes lists the scopes the caller would need to succeed, when
	// the backend provides this information.
	RequiredScopes []string
}

// NewAuthorizationError constructs an AuthorizationError.
func NewAuthorizationError(statusCode int, code, message, requestID, operationID string, requiredScopes []string) *AuthorizationError {
	return &AuthorizationError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
		RequiredScopes: requiredScopes,
	}
}

// Error implements the error interface.
func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s)",
		ErrAuthorization.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *AuthorizationError) Unwrap() error { return ErrAuthorization }

// ValidationError represents a 400 or 422 validation failure. It carries
// field-level details when the backend provides them.
type ValidationError struct {
	apiErrorBase
	// FieldErrors lists the individual field-level validation failures.
	FieldErrors []FieldError
}

// NewValidationError constructs a ValidationError.
func NewValidationError(statusCode int, code, message, requestID, operationID string, fieldErrors []FieldError) *ValidationError {
	return &ValidationError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
		FieldErrors: fieldErrors,
	}
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s, fields=%d)",
		ErrValidation.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID, len(e.FieldErrors))
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *ValidationError) Unwrap() error { return ErrValidation }

// NotFoundError represents a 404 resource-not-found response.
type NotFoundError struct {
	apiErrorBase
	// ResourceType identifies the kind of resource that was not found.
	ResourceType string
	// ResourceID identifies the specific resource that was not found.
	ResourceID string
}

// NewNotFoundError constructs a NotFoundError.
func NewNotFoundError(statusCode int, code, message, requestID, operationID, resourceType, resourceID string) *NotFoundError {
	return &NotFoundError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s, type=%s, id=%s)",
		ErrNotFound.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID, e.ResourceType, e.ResourceID)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *NotFoundError) Unwrap() error { return ErrNotFound }

// ConflictError represents a 409 resource conflict.
type ConflictError struct {
	apiErrorBase
}

// NewConflictError constructs a ConflictError.
func NewConflictError(statusCode int, code, message, requestID, operationID string) *ConflictError {
	return &ConflictError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
	}
}

// Error implements the error interface.
func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s)",
		ErrConflict.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *ConflictError) Unwrap() error { return ErrConflict }

// RateLimitError represents a 429 rate-limit response. RetryAfter carries the
// Retry-After hint from the server, if present.
type RateLimitError struct {
	apiErrorBase
	// RetryAfter is the server-suggested delay before retrying.
	RetryAfter time.Duration
}

// NewRateLimitError constructs a RateLimitError.
func NewRateLimitError(statusCode int, code, message, requestID, operationID string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
		RetryAfter: retryAfter,
	}
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s, retry_after=%s)",
		ErrRateLimit.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID, e.RetryAfter)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *RateLimitError) Unwrap() error { return ErrRateLimit }

// ServerError represents a 5xx server-side failure.
type ServerError struct {
	apiErrorBase
}

// NewServerError constructs a ServerError.
func NewServerError(statusCode int, code, message, requestID, operationID string) *ServerError {
	return &ServerError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
	}
}

// Error implements the error interface.
func (e *ServerError) Error() string {
	return fmt.Sprintf("%s: %s (status=%d, code=%s, request_id=%s, operation=%s)",
		ErrServer.Error(), e.message, e.statusCode, e.errorCode, e.requestID, e.operationID)
}

// Unwrap returns the sentinel so that errors.Is works.
func (e *ServerError) Unwrap() error { return ErrServer }

// UnexpectedError represents an error response that does not map to any known
// sentinel. CauseBody is capped at 4 KiB for safety. Use NewUnexpectedError
// to construct instances; it enforces the size cap.
type UnexpectedError struct {
	apiErrorBase
	// CauseBody is the raw response body, capped at 4 KiB.
	CauseBody []byte
}

// NewUnexpectedError constructs an UnexpectedError, capping causeBody at 4 KiB.
func NewUnexpectedError(statusCode int, code, message, requestID, operationID string, causeBody []byte) *UnexpectedError {
	var body []byte
	if causeBody != nil {
		if len(causeBody) > maxCauseBodyBytes {
			body = make([]byte, maxCauseBodyBytes)
			copy(body, causeBody[:maxCauseBodyBytes])
		} else {
			body = make([]byte, len(causeBody))
			copy(body, causeBody)
		}
	}
	return &UnexpectedError{
		apiErrorBase: apiErrorBase{
			statusCode:  statusCode,
			errorCode:   code,
			message:     message,
			requestID:   requestID,
			operationID: operationID,
		},
		CauseBody: body,
	}
}

// Error implements the error interface.
func (e *UnexpectedError) Error() string {
	return fmt.Sprintf("intelligencecloud: unexpected error: %s (status=%d, code=%s, request_id=%s, operation=%s)",
		e.message, e.statusCode, e.errorCode, e.requestID, e.operationID)
}

// Unwrap is not provided because UnexpectedError does not map to any sentinel.
// Callers should use errors.As to obtain this type directly.

// ---------------------------------------------------------------------------
// ConfigurationError — NOT an APIError
// ---------------------------------------------------------------------------

// ConfigurationError represents a configuration validation failure. It is
// returned only from option validation and client construction, never from API
// calls. It deliberately does NOT implement APIError.
type ConfigurationError struct {
	// Message describes the configuration problem.
	Message string
}

// Error implements the error interface.
func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("intelligencecloud: configuration error: %s", e.Message)
}
