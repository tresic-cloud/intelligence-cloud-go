// Package confirm provides destructive-verb preview rendering and
// confirmation prompting for the icctl CLI.
package confirm

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// EnvironmentInfo holds environment details for the preview.
type EnvironmentInfo struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

// DestructivePreview is the structured plan emitted before destructive verbs.
type DestructivePreview struct {
	Environment  EnvironmentInfo   `json:"environment"`
	Operation    string            `json:"operation"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Irreversible bool              `json:"irreversible"`
}

// RenderTablePreview writes a human-readable preview to w (typically stderr).
func RenderTablePreview(w io.Writer, p DestructivePreview) error {
	fmt.Fprintln(w, "The following operation is about to be performed:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Environment:    %s (%s)\n", p.Environment.Name, p.Environment.BaseURL)
	fmt.Fprintf(w, "  Operation:      %s\n", p.Operation)
	fmt.Fprintf(w, "  Resource type:  %s\n", p.ResourceType)
	fmt.Fprintf(w, "  Resource id:    %s\n", p.ResourceID)

	irreversibleStr := "no"
	if p.Irreversible {
		irreversibleStr = "yes"
	}
	fmt.Fprintf(w, "  Irreversible:   %s\n", irreversibleStr)

	if len(p.Attributes) > 0 {
		fmt.Fprintln(w, "  Attributes:")
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(p.Attributes))
		for k := range p.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "    %-14s%s\n", k+":", p.Attributes[k])
		}
	}
	fmt.Fprintln(w)
	return nil
}

// planEnvelope wraps the preview in a "plan" key for JSON output.
type planEnvelope struct {
	Plan DestructivePreview `json:"plan"`
}

// RenderJSONPreview writes a structured JSON preview to w (typically stdout).
func RenderJSONPreview(w io.Writer, p DestructivePreview) error {
	env := planEnvelope{Plan: p}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("json encode preview: %w", err)
	}
	return nil
}
