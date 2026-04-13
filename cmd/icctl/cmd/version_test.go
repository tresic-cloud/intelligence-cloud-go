package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/version"
)

func TestVersionCmd_TableOutput(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify version fields are present.
	checks := []struct {
		label string
		value string
	}{
		{"Version:", version.Version},
		{"Commit:", version.Commit},
		{"Built:", version.Date},
		{"Go version:", runtime.Version()},
		{"OS/Arch:", runtime.GOOS + "/" + runtime.GOARCH},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.label) {
			t.Errorf("expected output to contain label %q", c.label)
		}
		if !strings.Contains(output, c.value) {
			t.Errorf("expected output to contain value %q for label %q", c.value, c.label)
		}
	}
}

func TestVersionCmd_JSONOutput(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--output", "json", "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var info VersionInfo
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	if info.Version != version.Version {
		t.Errorf("expected version %q, got %q", version.Version, info.Version)
	}
	if info.Commit != version.Commit {
		t.Errorf("expected commit %q, got %q", version.Commit, info.Commit)
	}
	if info.Date != version.Date {
		t.Errorf("expected date %q, got %q", version.Date, info.Date)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("expected go version %q, got %q", runtime.Version(), info.GoVersion)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("expected OS %q, got %q", runtime.GOOS, info.OS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("expected arch %q, got %q", runtime.GOARCH, info.Arch)
	}
}

func TestVersionCmd_ExitCodeIsZero(t *testing.T) {
	cmd := NewRootCmd(nil, nil)
	cmd.SetContext(context.Background())
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected exit code 0 (no error), got error: %v", err)
	}
}

func TestVersionCmd_HasRequiredFields(t *testing.T) {
	// Verify the VersionInfo struct has all required JSON fields.
	info := VersionInfo{
		Version:   "v0.1.0",
		Commit:    "abc1234",
		Date:      "2026-04-13T10:00:00Z",
		GoVersion: "go1.23.0",
		OS:        "darwin",
		Arch:      "arm64",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expectedKeys := []string{"version", "commit", "date", "go_version", "os", "arch"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("expected JSON key %q in version output", key)
		}
	}
}

func TestVersionCmd_DefaultsAreDev(t *testing.T) {
	// When not built with ldflags, version defaults should indicate dev build.
	// This tests the current state of the version package.
	if version.Version == "" {
		t.Error("version.Version should not be empty")
	}
	if version.Commit == "" {
		t.Error("version.Commit should not be empty")
	}
	if version.Date == "" {
		t.Error("version.Date should not be empty")
	}
}
