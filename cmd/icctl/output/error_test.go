package output_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

func TestExitCode_Nil(t *testing.T) {
	if got := output.ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d; want 0", got)
	}
}

func TestExitCode_Authentication(t *testing.T) {
	err := ic.NewAuthenticationError(401, "token_expired", "expired", "req1", "GetMe")
	if got := output.ExitCode(err); got != 4 {
		t.Errorf("ExitCode(AuthenticationError) = %d; want 4", got)
	}
}

func TestExitCode_Authorization(t *testing.T) {
	err := ic.NewAuthorizationError(403, "forbidden", "not allowed", "req2", "DeleteReseller", nil)
	if got := output.ExitCode(err); got != 5 {
		t.Errorf("ExitCode(AuthorizationError) = %d; want 5", got)
	}
}

func TestExitCode_RateLimit(t *testing.T) {
	err := ic.NewRateLimitError(429, "rate_limited", "slow down", "req3", "ListResellers", 0)
	if got := output.ExitCode(err); got != 6 {
		t.Errorf("ExitCode(RateLimitError) = %d; want 6", got)
	}
}

func TestExitCode_Server(t *testing.T) {
	err := ic.NewServerError(500, "internal", "something broke", "req4", "GetCompany")
	if got := output.ExitCode(err); got != 7 {
		t.Errorf("ExitCode(ServerError) = %d; want 7", got)
	}
}

func TestExitCode_Configuration(t *testing.T) {
	err := &ic.ConfigurationError{Message: "bad config"}
	if got := output.ExitCode(err); got != 3 {
		t.Errorf("ExitCode(ConfigurationError) = %d; want 3", got)
	}
}

func TestExitCode_Validation(t *testing.T) {
	err := ic.NewValidationError(400, "invalid", "bad field", "req5", "CreateReseller", nil)
	if got := output.ExitCode(err); got != 1 {
		t.Errorf("ExitCode(ValidationError) = %d; want 1 (generic)", got)
	}
}

func TestExitCode_NotFound(t *testing.T) {
	err := ic.NewNotFoundError(404, "not_found", "gone", "req6", "GetReseller", "reseller", "rsl_123")
	if got := output.ExitCode(err); got != 1 {
		t.Errorf("ExitCode(NotFoundError) = %d; want 1 (generic)", got)
	}
}

func TestExitCode_Conflict(t *testing.T) {
	err := ic.NewConflictError(409, "conflict", "already exists", "req7", "CreateReseller")
	if got := output.ExitCode(err); got != 1 {
		t.Errorf("ExitCode(ConflictError) = %d; want 1 (generic)", got)
	}
}

func TestExitCode_ContextCanceled(t *testing.T) {
	if got := output.ExitCode(context.Canceled); got != 130 {
		t.Errorf("ExitCode(context.Canceled) = %d; want 130", got)
	}
}

func TestExitCode_ContextDeadline(t *testing.T) {
	if got := output.ExitCode(context.DeadlineExceeded); got != 124 {
		t.Errorf("ExitCode(context.DeadlineExceeded) = %d; want 124", got)
	}
}

func TestExitCode_NonTTYRefused(t *testing.T) {
	if got := output.ExitCode(output.ErrNonTTYRefused); got != 2 {
		t.Errorf("ExitCode(ErrNonTTYRefused) = %d; want 2", got)
	}
}

func TestExitCode_WrappedNonTTYRefused(t *testing.T) {
	wrapped := fmt.Errorf("operation failed: %w", output.ErrNonTTYRefused)
	if got := output.ExitCode(wrapped); got != 2 {
		t.Errorf("ExitCode(wrapped ErrNonTTYRefused) = %d; want 2", got)
	}
}

func TestExitCode_GenericError(t *testing.T) {
	err := errors.New("something else")
	if got := output.ExitCode(err); got != 1 {
		t.Errorf("ExitCode(generic) = %d; want 1", got)
	}
}

func TestExitCode_WrappedSentinels(t *testing.T) {
	// Wrapped authentication sentinel.
	wrapped := fmt.Errorf("outer: %w", ic.NewAuthenticationError(401, "x", "y", "z", "Op"))
	if got := output.ExitCode(wrapped); got != 4 {
		t.Errorf("ExitCode(wrapped AuthenticationError) = %d; want 4", got)
	}
}

func TestErrorInfoFromError_APIError(t *testing.T) {
	err := ic.NewAuthenticationError(401, "token_expired", "expired", "req_abc", "GetMe")
	info := output.ErrorInfoFromError(err)

	if info.Kind != "authentication" {
		t.Errorf("Kind = %q; want %q", info.Kind, "authentication")
	}
	if info.Status != 401 {
		t.Errorf("Status = %d; want 401", info.Status)
	}
	if info.RequestID != "req_abc" {
		t.Errorf("RequestID = %q; want %q", info.RequestID, "req_abc")
	}
}

func TestErrorInfoFromError_ConfigurationError(t *testing.T) {
	err := &ic.ConfigurationError{Message: "bad config"}
	info := output.ErrorInfoFromError(err)

	if info.Kind != "configuration" {
		t.Errorf("Kind = %q; want %q", info.Kind, "configuration")
	}
}

func TestErrorInfoFromError_GenericError(t *testing.T) {
	err := errors.New("unknown issue")
	info := output.ErrorInfoFromError(err)

	if info.Kind != "unexpected" {
		t.Errorf("Kind = %q; want %q", info.Kind, "unexpected")
	}
	if info.Message != "unknown issue" {
		t.Errorf("Message = %q; want %q", info.Message, "unknown issue")
	}
}

func TestErrorInfoFromError_Nil(t *testing.T) {
	info := output.ErrorInfoFromError(nil)
	if info.Kind != "" {
		t.Errorf("Kind = %q; want empty", info.Kind)
	}
}

func TestErrorKind_AllSentinels(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ic.NewValidationError(400, "", "", "", "", nil), "validation"},
		{ic.NewNotFoundError(404, "", "", "", "", "", ""), "not_found"},
		{ic.NewConflictError(409, "", "", "", ""), "conflict"},
		{ic.NewRateLimitError(429, "", "", "", "", 0), "rate_limit"},
		{ic.NewServerError(500, "", "", "", ""), "server"},
		{ic.NewAuthorizationError(403, "", "", "", "", nil), "authorization"},
	}

	for _, tt := range tests {
		info := output.ErrorInfoFromError(tt.err)
		if info.Kind != tt.want {
			t.Errorf("ErrorInfoFromError(%T).Kind = %q; want %q", tt.err, info.Kind, tt.want)
		}
	}
}
