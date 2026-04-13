package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// walkLeaves visits every leaf command (no subcommands of its own) in the
// cobra command tree rooted at cmd.
func walkLeaves(cmd *cobra.Command, visit func(*cobra.Command)) {
	for _, c := range cmd.Commands() {
		if len(c.Commands()) == 0 {
			visit(c)
		} else {
			walkLeaves(c, visit)
		}
	}
}

// isLongSentence checks that the Long description contains at least one
// sentence -- at minimum 10 characters plus a period or newline-delimited
// paragraph.
func isLongSentence(long string) bool {
	if len(long) < 10 {
		return false
	}
	// Contains a period (sentence ending) or a newline with content.
	if strings.Contains(long, ".") {
		return true
	}
	// Alternatively, contains a newline with at least 10 chars before it.
	if idx := strings.Index(long, "\n"); idx >= 10 {
		return true
	}
	return false
}

// TestHelpContract_AllLeaves walks the entire icctl command tree and verifies
// that every leaf subcommand satisfies FR-015 help contract:
//   - Use is non-empty
//   - Short is non-empty
//   - Long is non-empty and contains at least one sentence
//   - Example is non-empty
//   - --help output includes a flags section
//
// This test uses t.Errorf per leaf so all violations are reported, not just
// the first. It WILL FAIL until sibling agents (W4-1, W4-2) add their
// subcommands with proper Long and Example fields.
func TestHelpContract_AllLeaves(t *testing.T) {
	root := NewRootCmd(nil, nil)

	leafCount := 0

	walkLeaves(root, func(c *cobra.Command) {
		leafCount++
		path := c.CommandPath()

		if c.Use == "" {
			t.Errorf("[%s] Use is empty", path)
		}

		if c.Short == "" {
			t.Errorf("[%s] Short is empty", path)
		}

		if c.Long == "" {
			t.Errorf("[%s] Long is empty (FR-015 requires a description)", path)
		} else if !isLongSentence(c.Long) {
			t.Errorf("[%s] Long is too short or missing a sentence (got %d chars, no period/newline): %q",
				path, len(c.Long), c.Long)
		}

		if c.Example == "" {
			t.Errorf("[%s] Example is empty (FR-015 requires at least one example)", path)
		}

		// Execute --help and check for flags section.
		var buf bytes.Buffer
		c.SetOut(&buf)
		c.SetErr(&buf)
		c.SetArgs([]string{"--help"})

		// Reset the RunE to avoid actual execution side effects. We want
		// cobra to render help output.
		if err := c.Help(); err != nil {
			t.Errorf("[%s] Help() returned error: %v", path, err)
			return
		}

		helpOutput := buf.String()
		if !strings.Contains(helpOutput, "Flags:") && !strings.Contains(helpOutput, "flags:") {
			t.Errorf("[%s] --help output does not contain a Flags section", path)
		}
	})

	if leafCount == 0 {
		t.Fatal("No leaf commands found in the command tree; the test is misconfigured")
	}

	t.Logf("Checked %d leaf commands for help contract compliance", leafCount)
}

// TestHelpContract_RootHasSubcommands ensures the root command has at least
// the expected top-level subcommands registered.
func TestHelpContract_RootHasSubcommands(t *testing.T) {
	root := NewRootCmd(nil, nil)

	required := []string{"version", "profile", "completion"}
	names := make(map[string]bool)
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}

	for _, name := range required {
		if !names[name] {
			t.Errorf("root command missing required subcommand %q", name)
		}
	}
}

// TestHelpContract_LongDescription_Quality checks that Long descriptions
// across the existing command tree have meaningful content, not just a
// repeat of Short.
func TestHelpContract_LongDescription_Quality(t *testing.T) {
	root := NewRootCmd(nil, nil)

	walkLeaves(root, func(c *cobra.Command) {
		path := c.CommandPath()

		if c.Long == "" || c.Short == "" {
			return // Other tests cover this.
		}

		// Long should not be identical to Short.
		if strings.TrimSpace(c.Long) == strings.TrimSpace(c.Short) {
			t.Errorf("[%s] Long is identical to Short; FR-015 requires a more detailed description", path)
		}
	})
}
