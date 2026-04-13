package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVerticalsCmd creates the verticals parent command. No operations are
// currently available in the SDK for vertical resources (see IDB-1354 for
// upcoming API coverage). This command exists as a scaffold that prints a
// helpful message directing users to the tracking issue.
func NewVerticalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verticals",
		Short: "Manage vertical resources",
		Long: `Manage vertical resources in the Intelligence Cloud platform.

No operations are currently available for verticals. Vertical CRUD endpoints
are tracked under IDB-1354 and will be added in a future release. Once the
upstream API is available, subcommands such as list, get, create, update,
and delete will appear here automatically.`,
		Example: `  # Show available vertical subcommands (none yet)
  icctl verticals --help`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "No operations currently available for verticals.")
			fmt.Fprintln(cmd.OutOrStdout(), "See IDB-1354 for upcoming API coverage.")
			return nil
		},
	}

	return cmd
}
