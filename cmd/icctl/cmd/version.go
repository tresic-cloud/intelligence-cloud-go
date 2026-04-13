package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/version"
)

// VersionInfo holds the structured version information for JSON output.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// NewVersionCmd creates the version subcommand that prints version, commit,
// build date, and Go runtime information in either table or JSON format.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the version, build commit, build date, Go runtime version, and platform of this icctl binary.",
		Example: `  # Print version in table format (default)
  icctl version

  # Print version as JSON
  icctl version --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd)
		},
	}
}

func runVersion(cmd *cobra.Command) error {
	info := VersionInfo{
		Version:   version.Version,
		Commit:    version.Commit,
		Date:      version.Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	// Determine output format from state or flag.
	outputFmt := "table"
	if state := StateFrom(cmd.Context()); state != nil {
		outputFmt = state.EffectiveOutput
	}

	switch outputFmt {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	default:
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Version:    %s\n", info.Version)
		fmt.Fprintf(w, "Commit:     %s\n", info.Commit)
		fmt.Fprintf(w, "Built:      %s\n", info.Date)
		fmt.Fprintf(w, "Go version: %s\n", info.GoVersion)
		fmt.Fprintf(w, "OS/Arch:    %s/%s\n", info.OS, info.Arch)
		return nil
	}
}
