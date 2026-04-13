package confirm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

// ErrUserDenied is returned when the user declines the confirmation prompt.
var ErrUserDenied = errors.New("operation cancelled by user")

// PromptOptions configures the confirmation prompt behavior.
type PromptOptions struct {
	// AssumeYes bypasses the prompt (--yes/-y flag).
	AssumeYes bool
	// AssumeYesEnv is the value of ICCTL_ASSUME_YES env var.
	AssumeYesEnv string
	// IsTTY indicates whether stdin is a terminal.
	IsTTY bool
	// Stdin is the reader for user input.
	Stdin io.Reader
	// Stderr is the writer for the prompt text.
	Stderr io.Writer
}

// Prompt asks the user for confirmation. It respects the --yes flag,
// ICCTL_ASSUME_YES env var, and TTY detection.
//
// Returns nil if the user confirms, ErrUserDenied if they decline,
// or output.ErrNonTTYRefused if non-TTY without bypass.
func Prompt(opts PromptOptions) error {
	// Check --yes flag first.
	if opts.AssumeYes {
		return nil
	}

	// Check ICCTL_ASSUME_YES env var (case-insensitive).
	if isAssumeYesEnv(opts.AssumeYesEnv) {
		return nil
	}

	// Non-TTY without bypass: refuse.
	if !opts.IsTTY {
		return output.ErrNonTTYRefused
	}

	// TTY: show prompt and read input.
	fmt.Fprint(opts.Stderr, "Proceed? (y/N): ")

	scanner := bufio.NewScanner(opts.Stdin)
	if !scanner.Scan() {
		// EOF or error — treat as denial.
		return ErrUserDenied
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	switch answer {
	case "y", "yes":
		return nil
	default:
		return ErrUserDenied
	}
}

// isAssumeYesEnv checks if the ICCTL_ASSUME_YES env var value is truthy.
// Valid truthy values (case-insensitive): "1", "true", "yes".
func isAssumeYesEnv(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// IsTerminal checks if a file descriptor refers to a terminal.
// This is a convenience wrapper — in production, callers pass the result
// of golang.org/x/term.IsTerminal(int(os.Stdin.Fd())) as PromptOptions.IsTTY.
// The actual terminal detection is done at the call site to avoid importing
// golang.org/x/term in this package's test dependencies.
