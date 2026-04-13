package intelligencecloud

import (
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// [A-7] Sentinel errors and APIError interface
// ---------------------------------------------------------------------------

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrAuthentication", ErrAuthentication},
		{"ErrAuthorization", ErrAuthorization},
		{"ErrValidation", ErrValidation},
		{"ErrNotFound", ErrNotFound},
		{"ErrConflict", ErrConflict},
		{"ErrRateLimit", ErrRateLimit},
		{"ErrServer", ErrServer},
	}

	for i, a := range sentinels {
		if a.err == nil {
			t.Fatalf("%s is nil", a.name)
		}
		for j, b := range sentinels {
			if i != j && errors.Is(a.err, b.err) {
				t.Errorf("%s should not match %s", a.name, b.name)
			}
		}
	}
}

func TestSentinelErrors_Messages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrAuthentication, "intelligencecloud: authentication failed"},
		{ErrAuthorization, "intelligencecloud: not authorized"},
		{ErrValidation, "intelligencecloud: validation failed"},
		{ErrNotFound, "intelligencecloud: resource not found"},
		{ErrConflict, "intelligencecloud: resource conflict"},
		{ErrRateLimit, "intelligencecloud: rate-limited"},
		{ErrServer, "intelligencecloud: server error"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestAPIError_Interface(t *testing.T) {
	// Verify the APIError interface has the expected methods by compiling
	// a type assertion against a known concrete type.
	// AuthenticationError must satisfy APIError.
	var ae APIError = NewAuthenticationError(401, "auth_failed", "bad token", "req-1", "GetMe")
	if ae.Status() != 401 {
		t.Errorf("Status() = %d, want 401", ae.Status())
	}
	if ae.Code() != "auth_failed" {
		t.Errorf("Code() = %q, want %q", ae.Code(), "auth_failed")
	}
	if ae.RequestID() != "req-1" {
		t.Errorf("RequestID() = %q, want %q", ae.RequestID(), "req-1")
	}
	if ae.Operation() != "GetMe" {
		t.Errorf("Operation() = %q, want %q", ae.Operation(), "GetMe")
	}
}

// ---------------------------------------------------------------------------
// [A-8] Concrete error types with errors.Is / errors.As
// ---------------------------------------------------------------------------

func TestConcreteErrors_ErrorsIs(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		sentinel error
	}{
		{"AuthenticationError", NewAuthenticationError(401, "", "bad", "", ""), ErrAuthentication},
		{"AuthorizationError", NewAuthorizationError(403, "", "forbidden", "", "", nil), ErrAuthorization},
		{"ValidationError", NewValidationError(400, "", "invalid", "", "", nil), ErrValidation},
		{"NotFoundError", NewNotFoundError(404, "", "missing", "", "", "", ""), ErrNotFound},
		{"ConflictError", NewConflictError(409, "", "conflict", "", ""), ErrConflict},
		{"RateLimitError", NewRateLimitError(429, "", "slow down", "", "", 0), ErrRateLimit},
		{"ServerError", NewServerError(500, "", "boom", "", ""), ErrServer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.sentinel) {
				t.Errorf("errors.Is(%T, %v) = false, want true", tc.err, tc.sentinel)
			}
		})
	}
}

func TestConcreteErrors_ErrorsAs(t *testing.T) {
	t.Run("AuthenticationError", func(t *testing.T) {
		var target *AuthenticationError
		err := error(NewAuthenticationError(401, "auth_failed", "bad", "r1", "Op"))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if target.Status() != 401 || target.Code() != "auth_failed" ||
			target.RequestID() != "r1" || target.Operation() != "Op" {
			t.Errorf("fields not populated correctly via methods")
		}
	})

	t.Run("AuthorizationError", func(t *testing.T) {
		var target *AuthorizationError
		err := error(NewAuthorizationError(403, "forbidden", "no access", "r2", "ListResellers",
			[]string{"resellers.read", "resellers.write"}))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if len(target.RequiredScopes) != 2 || target.RequiredScopes[0] != "resellers.read" {
			t.Errorf("RequiredScopes = %v, want [resellers.read, resellers.write]", target.RequiredScopes)
		}
	})

	t.Run("NotFoundError", func(t *testing.T) {
		var target *NotFoundError
		err := error(NewNotFoundError(404, "not_found", "gone", "r3", "GetReseller", "reseller", "abc-123"))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if target.ResourceType != "reseller" || target.ResourceID != "abc-123" {
			t.Errorf("ResourceType=%q ResourceID=%q", target.ResourceType, target.ResourceID)
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		var target *RateLimitError
		err := error(NewRateLimitError(429, "rate_limited", "slow down", "r4", "Create", 30*time.Second))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if target.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", target.RetryAfter)
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		var target *ValidationError
		err := error(NewValidationError(422, "validation_failed", "bad input", "r5", "CreateReseller",
			[]FieldError{{Field: "name", Code: "required", Message: "name is required"}}))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if len(target.FieldErrors) != 1 || target.FieldErrors[0].Field != "name" {
			t.Errorf("FieldErrors = %+v", target.FieldErrors)
		}
	})

	t.Run("ConflictError", func(t *testing.T) {
		var target *ConflictError
		err := error(NewConflictError(409, "conflict", "already exists", "r6", "Create"))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if target.Status() != 409 {
			t.Errorf("Status() = %d, want 409", target.Status())
		}
	})

	t.Run("ServerError", func(t *testing.T) {
		var target *ServerError
		err := error(NewServerError(500, "internal", "boom", "r7", "Update"))
		if !errors.As(err, &target) {
			t.Fatal("errors.As failed")
		}
		if target.Status() != 500 {
			t.Errorf("Status() = %d, want 500", target.Status())
		}
	})
}

func TestUnexpectedError_CauseBodyCap(t *testing.T) {
	bigBody := make([]byte, 8192)
	for i := range bigBody {
		bigBody[i] = byte(i % 256)
	}
	ue := NewUnexpectedError(999, "unexpected", "something weird", "r8", "Op", bigBody)
	if len(ue.CauseBody) != 4096 {
		t.Errorf("CauseBody len = %d, want 4096", len(ue.CauseBody))
	}
	// Verify it is the first 4 KiB.
	for i := 0; i < 4096; i++ {
		if ue.CauseBody[i] != byte(i%256) {
			t.Fatalf("CauseBody[%d] = %d, want %d", i, ue.CauseBody[i], byte(i%256))
		}
	}
}

func TestUnexpectedError_SmallBodyNotTruncated(t *testing.T) {
	body := []byte("small body")
	ue := NewUnexpectedError(999, "unexpected", "something weird", "r8", "Op", body)
	if len(ue.CauseBody) != len(body) {
		t.Errorf("CauseBody len = %d, want %d", len(ue.CauseBody), len(body))
	}
}

func TestUnexpectedError_NilBody(t *testing.T) {
	ue := NewUnexpectedError(999, "unexpected", "something weird", "r8", "Op", nil)
	if ue.CauseBody != nil {
		t.Errorf("CauseBody = %v, want nil", ue.CauseBody)
	}
}

func TestUnexpectedError_ImplementsAPIError(t *testing.T) {
	var ae APIError = NewUnexpectedError(999, "x", "y", "z", "w", nil)
	if ae.Status() != 999 {
		t.Errorf("Status() = %d, want 999", ae.Status())
	}
}

func TestConcreteErrors_AllImplementAPIError(t *testing.T) {
	concretes := []APIError{
		NewAuthenticationError(401, "", "", "", ""),
		NewAuthorizationError(403, "", "", "", "", nil),
		NewValidationError(400, "", "", "", "", nil),
		NewNotFoundError(404, "", "", "", "", "", ""),
		NewConflictError(409, "", "", "", ""),
		NewRateLimitError(429, "", "", "", "", 0),
		NewServerError(500, "", "", "", ""),
		NewUnexpectedError(999, "", "", "", "", nil),
	}
	for _, c := range concretes {
		if c.Status() == 0 {
			t.Errorf("%T.Status() = 0", c)
		}
	}
}

func TestConcreteErrors_ErrorMessageNonEmpty(t *testing.T) {
	e := NewAuthenticationError(401, "auth_failed", "bad token", "r1", "GetMe")
	msg := e.Error()
	if msg == "" {
		t.Error("Error() returned empty string")
	}
}

// ---------------------------------------------------------------------------
// [A-9] ConfigurationError is NOT APIError
// ---------------------------------------------------------------------------

func TestConfigurationError_ImplementsError(t *testing.T) {
	ce := &ConfigurationError{Message: "bad config"}
	var e error = ce
	if e.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestConfigurationError_NotAPIError(t *testing.T) {
	ce := &ConfigurationError{Message: "bad config"}
	var ae APIError
	if errors.As(ce, &ae) {
		t.Error("ConfigurationError should NOT satisfy APIError")
	}
	// Also verify via direct type assertion.
	_, ok := interface{}(ce).(APIError)
	if ok {
		t.Error("ConfigurationError should NOT satisfy APIError via type assertion")
	}
}

// ---------------------------------------------------------------------------
// [A-28] FieldError round-trip on ValidationError
// ---------------------------------------------------------------------------

func TestFieldError_RoundTrip(t *testing.T) {
	fieldErrors := []FieldError{
		{Field: "name", Code: "required", Message: "name is required"},
		{Field: "email", Code: "invalid_format", Message: "must be a valid email"},
		{Field: "age", Code: "out_of_range", Message: "must be between 0 and 150"},
	}
	ve := NewValidationError(422, "validation_failed", "request validation failed",
		"req-abc", "CreateUser", fieldErrors)

	// Round-trip through errors.As.
	var target *ValidationError
	if !errors.As(error(ve), &target) {
		t.Fatal("errors.As failed for ValidationError")
	}
	if len(target.FieldErrors) != 3 {
		t.Fatalf("FieldErrors len = %d, want 3", len(target.FieldErrors))
	}
	for i, fe := range target.FieldErrors {
		if fe.Field != fieldErrors[i].Field {
			t.Errorf("FieldErrors[%d].Field = %q, want %q", i, fe.Field, fieldErrors[i].Field)
		}
		if fe.Code != fieldErrors[i].Code {
			t.Errorf("FieldErrors[%d].Code = %q, want %q", i, fe.Code, fieldErrors[i].Code)
		}
		if fe.Message != fieldErrors[i].Message {
			t.Errorf("FieldErrors[%d].Message = %q, want %q", i, fe.Message, fieldErrors[i].Message)
		}
	}

	// Also ensure errors.Is routes to the sentinel.
	if !errors.Is(ve, ErrValidation) {
		t.Error("errors.Is(ValidationError, ErrValidation) = false, want true")
	}
}
