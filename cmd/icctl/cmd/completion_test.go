package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot builds a minimal cobra command tree that mirrors the real icctl
// root just enough for completion generation to work.
func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "icctl",
		Short: "Intelligence Cloud CLI",
	}
	RegisterCompletionCmd(root)
	return root
}

// shellTestCase defines one entry in the table-driven completion tests.
type shellTestCase struct {
	shell  string
	marker string // substring that must appear in the generated output
}

var shellTests = []shellTestCase{
	{shell: "bash", marker: "_icctl"},
	{shell: "zsh", marker: "#compdef"},
	{shell: "fish", marker: "complete -c icctl"},
	{shell: "powershell", marker: "Register-ArgumentCompleter"},
}

func TestCompletionOutputNonEmpty(t *testing.T) {
	t.Parallel()
	for _, tc := range shellTests {
		t.Run(tc.shell, func(t *testing.T) {
			t.Parallel()
			root := newTestRoot()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetArgs([]string{"completion", tc.shell})
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s returned error: %v", tc.shell, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("completion %s produced empty output", tc.shell)
			}
		})
	}
}

func TestCompletionShellMarkers(t *testing.T) {
	t.Parallel()
	for _, tc := range shellTests {
		t.Run(tc.shell, func(t *testing.T) {
			t.Parallel()
			root := newTestRoot()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetArgs([]string{"completion", tc.shell})
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s returned error: %v", tc.shell, err)
			}
			output := buf.String()
			if !strings.Contains(output, tc.marker) {
				t.Errorf("completion %s output missing marker %q; got:\n%s",
					tc.shell, tc.marker, output[:min(len(output), 500)])
			}
		})
	}
}

func TestCompletionExitZero(t *testing.T) {
	t.Parallel()
	for _, tc := range shellTests {
		t.Run(tc.shell, func(t *testing.T) {
			t.Parallel()
			root := newTestRoot()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetArgs([]string{"completion", tc.shell})
			err := root.Execute()
			if err != nil {
				t.Errorf("completion %s: expected exit 0, got error: %v", tc.shell, err)
			}
		})
	}
}

func TestCompletionNoArg(t *testing.T) {
	t.Parallel()
	root := newTestRoot()
	var buf bytes.Buffer
	root.SetErr(&buf)
	root.SetArgs([]string{"completion"})
	err := root.Execute()
	if err == nil {
		t.Fatal("completion with no args should return an error")
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	t.Parallel()
	root := newTestRoot()
	var buf bytes.Buffer
	root.SetErr(&buf)
	root.SetArgs([]string{"completion", "unknownShell"})
	err := root.Execute()
	if err == nil {
		t.Fatal("completion with invalid shell should return an error")
	}
	if !strings.Contains(err.Error(), "invalid argument") {
		t.Errorf("expected 'invalid argument' in error, got: %v", err)
	}
}

// shellParseCheck defines a parse-validation test using the shell's own parser.
type shellParseCheck struct {
	shell   string
	command string
	args    []string
}

var parseChecks = []shellParseCheck{
	{shell: "bash", command: "bash", args: []string{"-n"}},
	{shell: "zsh", command: "zsh", args: []string{"-n"}},
	{shell: "fish", command: "fish", args: []string{"--no-execute"}},
}

func TestCompletionShellParseable(t *testing.T) {
	t.Parallel()
	for _, pc := range parseChecks {
		t.Run(pc.shell, func(t *testing.T) {
			t.Parallel()

			// Skip if the shell binary is not available.
			path, err := exec.LookPath(pc.command)
			if err != nil {
				t.Skipf("%s not found in PATH; skipping parse check", pc.command)
			}

			// Generate the completion script.
			root := newTestRoot()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetArgs([]string{"completion", pc.shell})
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s returned error: %v", pc.shell, err)
			}

			// Write to a temp file and parse-check via the shell.
			tmpFile, err := os.CreateTemp("", "icctl-completion-*."+pc.shell)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(buf.Bytes()); err != nil {
				tmpFile.Close()
				t.Fatalf("failed to write temp file: %v", err)
			}
			tmpFile.Close()

			args := append(pc.args, tmpFile.Name())
			cmd := exec.Command(path, args...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Errorf("%s parse check failed: %v\nstderr: %s", pc.shell, err, stderr.String())
			}
		})
	}
}

func TestRegisterCompletionCmd(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "icctl"}
	RegisterCompletionCmd(root)

	found := false
	for _, c := range root.Commands() {
		if c.Name() == "completion" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("RegisterCompletionCmd did not add 'completion' subcommand to root")
	}
}

func TestRegisterCompletionCmdValidArgs(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "icctl"}
	RegisterCompletionCmd(root)

	var completionCmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "completion" {
			completionCmd = c
			break
		}
	}
	if completionCmd == nil {
		t.Fatal("completion subcommand not found")
	}

	expected := []string{"bash", "zsh", "fish", "powershell"}
	if len(completionCmd.ValidArgs) != len(expected) {
		t.Fatalf("expected %d valid args, got %d", len(expected), len(completionCmd.ValidArgs))
	}
	for i, want := range expected {
		if completionCmd.ValidArgs[i] != want {
			t.Errorf("ValidArgs[%d] = %q, want %q", i, completionCmd.ValidArgs[i], want)
		}
	}
}
