package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONFormatter writes output as JSON.
type JSONFormatter struct {
	stdout io.Writer
}

// NewJSONFormatter creates a JSONFormatter that writes to the given writer.
func NewJSONFormatter(stdout io.Writer) *JSONFormatter {
	return &JSONFormatter{stdout: stdout}
}

// Format returns "json".
func (f *JSONFormatter) Format() string { return "json" }

// WriteEntity writes a single entity as a JSON document to stdout.
func (f *JSONFormatter) WriteEntity(entity interface{}) error {
	return writeJSON(f.stdout, entity)
}

// listResponse is the envelope for list results.
type listResponse struct {
	Items    []interface{} `json:"items"`
	PageInfo PageInfo      `json:"page_info"`
}

// WriteList writes a list of items with pagination info as a JSON document.
func (f *JSONFormatter) WriteList(items []interface{}, pageInfo PageInfo) error {
	if items == nil {
		items = []interface{}{}
	}
	resp := listResponse{
		Items:    items,
		PageInfo: pageInfo,
	}
	return writeJSON(f.stdout, resp)
}

// errorResponse is the envelope for error output.
type errorResponse struct {
	Error ErrorInfo `json:"error"`
}

// WriteError writes an error as a JSON document to stderr.
func (f *JSONFormatter) WriteError(stderr io.Writer, errInfo ErrorInfo) error {
	resp := errorResponse{Error: errInfo}
	return writeJSON(stderr, resp)
}

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}
