package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

func TestTableFormatter_WriteEntity(t *testing.T) {
	var buf bytes.Buffer
	f := output.NewTableFormatter(&buf, true) // UTF-8 enabled

	headers := []string{"ID", "Name", "Active"}
	rows := [][]string{
		{"rsl_01HW3XYZ", "Acme Bakery", "true"},
	}

	if err := f.WriteTable(headers, rows); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "rsl_01HW3XYZ") {
		t.Errorf("output missing ID: %s", out)
	}
	if !strings.Contains(out, "Acme Bakery") {
		t.Errorf("output missing Name: %s", out)
	}
}

func TestTableFormatter_BoolUTF8(t *testing.T) {
	if got := output.FormatBool(true, true); got != "\u2713" {
		t.Errorf("FormatBool(true, utf8=true) = %q; want ✓", got)
	}
	if got := output.FormatBool(false, true); got != "\u2717" {
		t.Errorf("FormatBool(false, utf8=true) = %q; want ✗", got)
	}
}

func TestTableFormatter_BoolASCII(t *testing.T) {
	if got := output.FormatBool(true, false); got != "Y" {
		t.Errorf("FormatBool(true, utf8=false) = %q; want Y", got)
	}
	if got := output.FormatBool(false, false); got != "N" {
		t.Errorf("FormatBool(false, utf8=false) = %q; want N", got)
	}
}

func TestTableFormatter_MultipleRows(t *testing.T) {
	var buf bytes.Buffer
	f := output.NewTableFormatter(&buf, true)

	headers := []string{"ID", "Name"}
	rows := [][]string{
		{"1", "Alpha"},
		{"2", "Beta"},
		{"3", "Gamma"},
	}

	if err := f.WriteTable(headers, rows); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	out := buf.String()
	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		if !strings.Contains(out, name) {
			t.Errorf("output missing %q: %s", name, out)
		}
	}
}

func TestTableFormatter_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	f := output.NewTableFormatter(&buf, true)

	headers := []string{"ID", "Name"}
	if err := f.WriteTable(headers, nil); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	out := buf.String()
	// Should still have headers.
	if !strings.Contains(out, "ID") {
		t.Errorf("output missing header: %s", out)
	}
}

func TestTableFormatter_ErrorToStderr(t *testing.T) {
	var stderr bytes.Buffer
	f := output.NewTableFormatter(nil, true) // stdout unused for errors

	errInfo := output.ErrorInfo{
		Kind:    "authentication",
		Message: "Bearer token expired",
	}

	if err := f.WriteErrorText(&stderr, errInfo); err != nil {
		t.Fatalf("WriteErrorText: %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "Bearer token expired") {
		t.Errorf("stderr missing error message: %s", out)
	}
	if !strings.Contains(out, "Error") {
		t.Errorf("stderr missing Error label: %s", out)
	}
}
