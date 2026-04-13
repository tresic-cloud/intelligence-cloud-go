package cmd

import (
	"context"
	"errors"
	"testing"
)

// TestSIGINT_ExitCode130 verifies that context cancellation (which is what
// SIGINT causes via signal.NotifyContext) results in the correct exit code.
// The actual SIGINT delivery is tested via subprocess in an integration test;
// here we verify the exit-code mapping logic.
func TestSIGINT_ExitCode130(t *testing.T) {
	err := context.Canceled
	exitCode := MapErrorToExitCode(err)
	if exitCode != ExitInterrupted {
		t.Errorf("expected exit code %d for context.Canceled, got %d", ExitInterrupted, exitCode)
	}
}

func TestTimeout_ExitCode124(t *testing.T) {
	err := context.DeadlineExceeded
	exitCode := MapErrorToExitCode(err)
	if exitCode != ExitTimeout {
		t.Errorf("expected exit code %d for context.DeadlineExceeded, got %d", ExitTimeout, exitCode)
	}
}

func TestGenericError_ExitCode1(t *testing.T) {
	err := errors.New("some generic error")
	exitCode := MapErrorToExitCode(err)
	if exitCode != ExitGenericError {
		t.Errorf("expected exit code %d for generic error, got %d", ExitGenericError, exitCode)
	}
}

func TestNilError_ExitCode0(t *testing.T) {
	exitCode := MapErrorToExitCode(nil)
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d for nil error, got %d", ExitSuccess, exitCode)
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify exit code constants match cli-schema.md.
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"Success", ExitSuccess, 0},
		{"GenericError", ExitGenericError, 1},
		{"UsageError", ExitUsageError, 2},
		{"ConfigError", ExitConfigError, 3},
		{"AuthNError", ExitAuthNError, 4},
		{"AuthZError", ExitAuthZError, 5},
		{"RateLimitErr", ExitRateLimitErr, 6},
		{"ServerError", ExitServerError, 7},
		{"Timeout", ExitTimeout, 124},
		{"Interrupted", ExitInterrupted, 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("expected %s=%d, got %d", tt.name, tt.expected, tt.code)
			}
		})
	}
}

func TestWrappedContextErrors(t *testing.T) {
	// Context errors wrapped in other errors should still be detected.
	wrappedCancel := errors.Join(errors.New("outer"), context.Canceled)
	if MapErrorToExitCode(wrappedCancel) != ExitInterrupted {
		t.Errorf("expected exit code %d for wrapped context.Canceled", ExitInterrupted)
	}

	wrappedDeadline := errors.Join(errors.New("outer"), context.DeadlineExceeded)
	if MapErrorToExitCode(wrappedDeadline) != ExitTimeout {
		t.Errorf("expected exit code %d for wrapped context.DeadlineExceeded", ExitTimeout)
	}
}
