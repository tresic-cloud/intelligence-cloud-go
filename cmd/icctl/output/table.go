package output

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
)

// TableFormatter writes output as formatted tables.
type TableFormatter struct {
	stdout io.Writer
	utf8   bool
}

// NewTableFormatter creates a TableFormatter. If utf8 is true, boolean
// values render as UTF-8 check/cross marks; otherwise ASCII Y/N.
func NewTableFormatter(stdout io.Writer, utf8 bool) *TableFormatter {
	return &TableFormatter{stdout: stdout, utf8: utf8}
}

// Format returns "table".
func (f *TableFormatter) Format() string { return "table" }

// WriteTable writes headers and rows as a formatted table to stdout.
func (f *TableFormatter) WriteTable(headers []string, rows [][]string) error {
	table := tablewriter.NewWriter(f.stdout)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
	return nil
}

// WriteErrorText writes a human-readable error to the given writer (stderr).
func (f *TableFormatter) WriteErrorText(stderr io.Writer, errInfo ErrorInfo) error {
	_, err := fmt.Fprintf(stderr, "Error: %s\n", errInfo.Message)
	if err != nil {
		return err
	}
	if errInfo.Kind != "" {
		_, err = fmt.Fprintf(stderr, "  Kind: %s\n", errInfo.Kind)
		if err != nil {
			return err
		}
	}
	if errInfo.Code != "" {
		_, err = fmt.Fprintf(stderr, "  Code: %s\n", errInfo.Code)
		if err != nil {
			return err
		}
	}
	if errInfo.RequestID != "" {
		_, err = fmt.Fprintf(stderr, "  Request ID: %s\n", errInfo.RequestID)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatBool formats a boolean for display. If utf8 is true, uses
// check/cross marks; otherwise Y/N.
func FormatBool(b bool, utf8 bool) string {
	if utf8 {
		if b {
			return "\u2713" // ✓
		}
		return "\u2717" // ✗
	}
	if b {
		return "Y"
	}
	return "N"
}
