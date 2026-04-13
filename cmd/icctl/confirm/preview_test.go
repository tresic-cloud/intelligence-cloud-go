package confirm_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/confirm"
)

func newTestPreview() confirm.DestructivePreview {
	return confirm.DestructivePreview{
		Environment: confirm.EnvironmentInfo{
			Name:    "production",
			BaseURL: "https://api.intelligence.cloud",
		},
		Operation:    "Delete reseller",
		ResourceType: "reseller",
		ResourceID:   "rsl_01HW3XYZ",
		Attributes: map[string]string{
			"name":   "Acme Bakery, Inc.",
			"status": "active",
		},
		Irreversible: true,
	}
}

func TestRenderTablePreview(t *testing.T) {
	p := newTestPreview()
	var stderr bytes.Buffer

	if err := confirm.RenderTablePreview(&stderr, p); err != nil {
		t.Fatalf("RenderTablePreview: %v", err)
	}

	out := stderr.String()

	// Check all required fields are present.
	checks := []string{
		"production",
		"https://api.intelligence.cloud",
		"Delete reseller",
		"reseller",
		"rsl_01HW3XYZ",
		"yes",
		"Acme Bakery, Inc.",
		"active",
		"The following operation is about to be performed",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("table preview missing %q:\n%s", s, out)
		}
	}
}

func TestRenderTablePreview_NotIrreversible(t *testing.T) {
	p := newTestPreview()
	p.Irreversible = false

	var stderr bytes.Buffer
	if err := confirm.RenderTablePreview(&stderr, p); err != nil {
		t.Fatalf("RenderTablePreview: %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "no") {
		t.Errorf("expected 'no' for Irreversible=false:\n%s", out)
	}
}

func TestRenderTablePreview_NoAttributes(t *testing.T) {
	p := newTestPreview()
	p.Attributes = nil

	var stderr bytes.Buffer
	if err := confirm.RenderTablePreview(&stderr, p); err != nil {
		t.Fatalf("RenderTablePreview: %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "Delete reseller") {
		t.Errorf("missing operation:\n%s", out)
	}
}

func TestRenderJSONPreview(t *testing.T) {
	p := newTestPreview()
	var stdout bytes.Buffer

	if err := confirm.RenderJSONPreview(&stdout, p); err != nil {
		t.Fatalf("RenderJSONPreview: %v", err)
	}

	var got struct {
		Plan confirm.DestructivePreview `json:"plan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\nraw: %s", err, stdout.String())
	}

	if got.Plan.Operation != "Delete reseller" {
		t.Errorf("operation = %q; want %q", got.Plan.Operation, "Delete reseller")
	}
	if got.Plan.Environment.Name != "production" {
		t.Errorf("environment.name = %q; want %q", got.Plan.Environment.Name, "production")
	}
	if got.Plan.Environment.BaseURL != "https://api.intelligence.cloud" {
		t.Errorf("environment.base_url = %q; want %q", got.Plan.Environment.BaseURL, "https://api.intelligence.cloud")
	}
	if got.Plan.ResourceType != "reseller" {
		t.Errorf("resource_type = %q; want %q", got.Plan.ResourceType, "reseller")
	}
	if got.Plan.ResourceID != "rsl_01HW3XYZ" {
		t.Errorf("resource_id = %q; want %q", got.Plan.ResourceID, "rsl_01HW3XYZ")
	}
	if !got.Plan.Irreversible {
		t.Error("irreversible should be true")
	}
	if got.Plan.Attributes["name"] != "Acme Bakery, Inc." {
		t.Errorf("attributes.name = %q; want %q", got.Plan.Attributes["name"], "Acme Bakery, Inc.")
	}
}

func TestRenderJSONPreview_Structure(t *testing.T) {
	p := newTestPreview()
	var stdout bytes.Buffer

	if err := confirm.RenderJSONPreview(&stdout, p); err != nil {
		t.Fatalf("RenderJSONPreview: %v", err)
	}

	// Verify top-level has "plan" key.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["plan"]; !ok {
		t.Error("JSON output missing top-level 'plan' key")
	}
}

func TestRenderTablePreview_WrittenToStderr(t *testing.T) {
	// Confirm that table preview goes to the provided writer (stderr).
	p := newTestPreview()
	var stderr bytes.Buffer

	if err := confirm.RenderTablePreview(&stderr, p); err != nil {
		t.Fatalf("RenderTablePreview: %v", err)
	}

	if stderr.Len() == 0 {
		t.Error("stderr should have content")
	}
}
