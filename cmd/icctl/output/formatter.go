// Package output provides output formatters for the icctl CLI, supporting
// JSON and table formats. Errors are always written to stderr.
package output

import (
	"fmt"
	"io"
)

// Formatter is the interface for output formatters.
type Formatter interface {
	// Format returns the format name ("json" or "table").
	Format() string
}

// PageInfo describes pagination state for list results.
type PageInfo struct {
	ItemsFetched  int    `json:"items_fetched"`
	PagesFetched  int    `json:"pages_fetched"`
	HasNextPage   bool   `json:"has_next_page"`
	NextPageToken string `json:"next_page_token"`
}

// ErrorInfo holds structured error information for output.
type ErrorInfo struct {
	Kind      string `json:"kind"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Operation string `json:"operation"`
}

// NewFormatter creates a Formatter for the given format name.
// stdout is the writer for normal output.
func NewFormatter(format string) (Formatter, error) {
	switch format {
	case "json":
		return NewJSONFormatter(nil), nil
	case "table":
		return NewTableFormatter(nil, true), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %q (valid: json, table)", format)
	}
}

// NewFormatterWithWriter creates a Formatter with a specific stdout writer.
func NewFormatterWithWriter(format string, stdout io.Writer, utf8 bool) (Formatter, error) {
	switch format {
	case "json":
		return NewJSONFormatter(stdout), nil
	case "table":
		return NewTableFormatter(stdout, utf8), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %q (valid: json, table)", format)
	}
}
