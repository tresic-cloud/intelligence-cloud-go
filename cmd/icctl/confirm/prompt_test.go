package confirm_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/confirm"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

func TestPrompt_YesBypass(t *testing.T) {
	opts := confirm.PromptOptions{
		AssumeYes: true,
		IsTTY:     false,
		Stdin:     strings.NewReader(""),
		Stderr:    &bytes.Buffer{},
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt with --yes: %v", err)
	}
}

func TestPrompt_EnvBypass_True(t *testing.T) {
	opts := confirm.PromptOptions{
		AssumeYes:   false,
		AssumeYesEnv: "true",
		IsTTY:       false,
		Stdin:       strings.NewReader(""),
		Stderr:      &bytes.Buffer{},
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt with ICCTL_ASSUME_YES=true: %v", err)
	}
}

func TestPrompt_EnvBypass_1(t *testing.T) {
	opts := confirm.PromptOptions{
		AssumeYes:   false,
		AssumeYesEnv: "1",
		IsTTY:       false,
		Stdin:       strings.NewReader(""),
		Stderr:      &bytes.Buffer{},
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt with ICCTL_ASSUME_YES=1: %v", err)
	}
}

func TestPrompt_EnvBypass_Yes(t *testing.T) {
	opts := confirm.PromptOptions{
		AssumeYes:   false,
		AssumeYesEnv: "yes",
		IsTTY:       false,
		Stdin:       strings.NewReader(""),
		Stderr:      &bytes.Buffer{},
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt with ICCTL_ASSUME_YES=yes: %v", err)
	}
}

func TestPrompt_EnvBypass_CaseInsensitive(t *testing.T) {
	for _, val := range []string{"TRUE", "True", "YES", "Yes", "1"} {
		opts := confirm.PromptOptions{
			AssumeYes:   false,
			AssumeYesEnv: val,
			IsTTY:       false,
			Stdin:       strings.NewReader(""),
			Stderr:      &bytes.Buffer{},
		}

		if err := confirm.Prompt(opts); err != nil {
			t.Errorf("Prompt with ICCTL_ASSUME_YES=%q: %v", val, err)
		}
	}
}

func TestPrompt_EnvBypass_InvalidValues(t *testing.T) {
	for _, val := range []string{"0", "false", "no", "maybe", ""} {
		opts := confirm.PromptOptions{
			AssumeYes:   false,
			AssumeYesEnv: val,
			IsTTY:       false,
			Stdin:       strings.NewReader(""),
			Stderr:      &bytes.Buffer{},
		}

		err := confirm.Prompt(opts)
		if err == nil {
			t.Errorf("Prompt with ICCTL_ASSUME_YES=%q: expected error, got nil", val)
		}
	}
}

func TestPrompt_NonTTY_NoBypass_ReturnsError(t *testing.T) {
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     false,
		Stdin:     strings.NewReader(""),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err == nil {
		t.Fatal("Prompt non-TTY without --yes: expected error, got nil")
	}
	if !errors.Is(err, output.ErrNonTTYRefused) {
		t.Errorf("error = %v; want ErrNonTTYRefused", err)
	}
}

func TestPrompt_TTY_UserConfirmsY(t *testing.T) {
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     true,
		Stdin:     strings.NewReader("y\n"),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt TTY y: %v", err)
	}

	// Check that the prompt was written to stderr.
	if !strings.Contains(stderr.String(), "Proceed?") {
		t.Errorf("stderr missing prompt: %s", stderr.String())
	}
}

func TestPrompt_TTY_UserConfirmsYes(t *testing.T) {
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     true,
		Stdin:     strings.NewReader("yes\n"),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err != nil {
		t.Fatalf("Prompt TTY yes: %v", err)
	}
}

func TestPrompt_TTY_UserDeniesN(t *testing.T) {
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     true,
		Stdin:     strings.NewReader("N\n"),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err == nil {
		t.Fatal("Prompt TTY N: expected error, got nil")
	}
}

func TestPrompt_TTY_UserDeniesEmpty(t *testing.T) {
	// Default is N, so empty input should deny.
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     true,
		Stdin:     strings.NewReader("\n"),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err == nil {
		t.Fatal("Prompt TTY empty: expected error (default N), got nil")
	}
}

func TestPrompt_TTY_UserDeniesNo(t *testing.T) {
	var stderr bytes.Buffer
	opts := confirm.PromptOptions{
		AssumeYes: false,
		IsTTY:     true,
		Stdin:     strings.NewReader("no\n"),
		Stderr:    &stderr,
	}

	err := confirm.Prompt(opts)
	if err == nil {
		t.Fatal("Prompt TTY no: expected error, got nil")
	}
}

func TestPrompt_TTY_CaseInsensitiveY(t *testing.T) {
	for _, input := range []string{"Y\n", "y\n", "YES\n", "Yes\n"} {
		var stderr bytes.Buffer
		opts := confirm.PromptOptions{
			IsTTY:  true,
			Stdin:  strings.NewReader(input),
			Stderr: &stderr,
		}

		if err := confirm.Prompt(opts); err != nil {
			t.Errorf("Prompt(%q): %v", strings.TrimSpace(input), err)
		}
	}
}
