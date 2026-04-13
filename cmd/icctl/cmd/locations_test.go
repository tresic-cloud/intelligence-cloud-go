package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLocationsCmd_RunPrintsScaffoldMessage(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.AddCommand(NewLocationsCmd())
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"locations"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No operations currently available") {
		t.Errorf("expected scaffold message, got: %s", output)
	}
	if !strings.Contains(output, "IDB-1354") {
		t.Errorf("expected IDB-1354 reference, got: %s", output)
	}
}

func TestLocationsCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.AddCommand(NewLocationsCmd())
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"locations", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "IDB-1354") {
		t.Errorf("expected IDB-1354 in help output, got: %s", output)
	}
}
