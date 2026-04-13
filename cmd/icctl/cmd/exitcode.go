package cmd

import (
	"context"
	"errors"
)

// Exit codes per cli-schema.md.
const (
	ExitSuccess       = 0
	ExitGenericError  = 1
	ExitUsageError    = 2
	ExitConfigError   = 3
	ExitAuthNError    = 4
	ExitAuthZError    = 5
	ExitRateLimitErr  = 6
	ExitServerError   = 7
	ExitTimeout       = 124
	ExitInterrupted   = 130
)

// MapErrorToExitCode classifies an error and returns the appropriate exit
// code per cli-schema.md. This is a basic mapper that handles context errors;
// the full SDK-error-aware mapper lives in cmd/icctl/output/error.go (W3-1).
func MapErrorToExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ExitTimeout
	}

	if errors.Is(err, context.Canceled) {
		return ExitInterrupted
	}

	return ExitGenericError
}
