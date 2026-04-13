package output_test

import (
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

func TestNewFormatter_JSON(t *testing.T) {
	f, err := output.NewFormatter("json")
	if err != nil {
		t.Fatalf("NewFormatter(json): %v", err)
	}
	if f == nil {
		t.Fatal("NewFormatter(json) returned nil")
	}
	if f.Format() != "json" {
		t.Errorf("Format() = %q; want %q", f.Format(), "json")
	}
}

func TestNewFormatter_Table(t *testing.T) {
	f, err := output.NewFormatter("table")
	if err != nil {
		t.Fatalf("NewFormatter(table): %v", err)
	}
	if f == nil {
		t.Fatal("NewFormatter(table) returned nil")
	}
	if f.Format() != "table" {
		t.Errorf("Format() = %q; want %q", f.Format(), "table")
	}
}

func TestNewFormatter_Invalid(t *testing.T) {
	_, err := output.NewFormatter("xml")
	if err == nil {
		t.Fatal("NewFormatter(xml): expected error, got nil")
	}
}

func TestNewFormatter_Empty(t *testing.T) {
	_, err := output.NewFormatter("")
	if err == nil {
		t.Fatal("NewFormatter(empty): expected error, got nil")
	}
}
