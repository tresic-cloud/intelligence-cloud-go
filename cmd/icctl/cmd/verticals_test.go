package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVerticalsCmd_RunPrintsScaffoldMessage(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.AddCommand(NewVerticalsCmd())
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"verticals"})

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

func TestVerticalsCmd_HelpDoesNotError(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.AddCommand(NewVerticalsCmd())
	root.SetContext(context.Background())

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"verticals", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "IDB-1354") {
		t.Errorf("expected IDB-1354 in help output, got: %s", output)
	}
}
