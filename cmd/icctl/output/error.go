package output

import (
	"context"
	"errors"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
)

// Exit codes per cli-schema.md.
const (
	ExitSuccess       = 0
	ExitGeneric       = 1
	ExitUsage         = 2
	ExitConfiguration = 3
	ExitAuth          = 4
	ExitAuthz         = 5
	ExitRateLimit     = 6
	ExitServer        = 7
	ExitTimeout       = 124
	ExitInterrupt     = 130
)

// ErrNonTTYRefused is returned when a destructive verb is invoked on a
// non-TTY without --yes or ICCTL_ASSUME_YES. It maps to exit code 2.
var ErrNonTTYRefused = errors.New("refusing destructive operation: not a TTY and --yes not specified")

// ExitCode maps an error to a CLI exit code using errors.Is and errors.As
// against the SDK's typed error hierarchy.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check for non-TTY refusal (usage error).
	if errors.Is(err, ErrNonTTYRefused) {
		return ExitUsage
	}

	// Check for context cancellation (SIGINT).
	if errors.Is(err, context.Canceled) {
		return ExitInterrupt
	}

	// Check for context deadline (timeout).
	if errors.Is(err, context.DeadlineExceeded) {
		return ExitTimeout
	}

	// Check SDK sentinel errors.
	if errors.Is(err, ic.ErrAuthentication) {
		return ExitAuth
	}
	if errors.Is(err, ic.ErrAuthorization) {
		return ExitAuthz
	}
	if errors.Is(err, ic.ErrRateLimit) {
		return ExitRateLimit
	}
	if errors.Is(err, ic.ErrServer) {
		return ExitServer
	}

	// Check for ConfigurationError.
	var cfgErr *ic.ConfigurationError
	if errors.As(err, &cfgErr) {
		return ExitConfiguration
	}

	// Validation, NotFound, Conflict, and other errors map to generic failure.
	return ExitGeneric
}

// ErrorInfoFromError extracts structured error info from an SDK error for
// formatted output. If the error does not implement APIError, a generic
// error info is returned.
func ErrorInfoFromError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{}
	}

	var apiErr ic.APIError
	if errors.As(err, &apiErr) {
		return ErrorInfo{
			Kind:      errorKind(err),
			Status:    apiErr.Status(),
			Code:      apiErr.Code(),
			Message:   err.Error(),
			RequestID: apiErr.RequestID(),
			Operation: apiErr.Operation(),
		}
	}

	var cfgErr *ic.ConfigurationError
	if errors.As(err, &cfgErr) {
		return ErrorInfo{
			Kind:    "configuration",
			Message: cfgErr.Error(),
		}
	}

	return ErrorInfo{
		Kind:    errorKind(err),
		Message: err.Error(),
	}
}

// errorKind returns the string classification of an error.
func errorKind(err error) string {
	switch {
	case errors.Is(err, ic.ErrAuthentication):
		return "authentication"
	case errors.Is(err, ic.ErrAuthorization):
		return "authorization"
	case errors.Is(err, ic.ErrValidation):
		return "validation"
	case errors.Is(err, ic.ErrNotFound):
		return "not_found"
	case errors.Is(err, ic.ErrConflict):
		return "conflict"
	case errors.Is(err, ic.ErrRateLimit):
		return "rate_limit"
	case errors.Is(err, ic.ErrServer):
		return "server"
	default:
		var cfgErr *ic.ConfigurationError
		if errors.As(err, &cfgErr) {
			return "configuration"
		}
		return "unexpected"
	}
}
