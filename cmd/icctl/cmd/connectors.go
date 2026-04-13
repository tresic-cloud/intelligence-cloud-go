package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/tui"
)

// NewConnectorsCmd creates the connectors command tree with list-by-location
// and list-by-company subcommands.
func NewConnectorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connectors",
		Short: "Manage connector resources",
		Long:  "Manage connector resources in the Intelligence Cloud platform. Connectors can be listed by location or by company.",
		Example: `  # List connectors for a location
  icctl connectors list-by-location loc_01HW3XYZ

  # List connectors for a company
  icctl connectors list-by-company cmp_01HW3XYZ`,
	}

	cmd.AddCommand(newConnectorsListByLocationCmd())
	cmd.AddCommand(newConnectorsListByCompanyCmd())

	return cmd
}

func newConnectorsListByLocationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-location <location-id>",
		Short: "List connectors for a location",
		Long:  "List connectors scoped to a specific location. Results are paginated and support filtering by search term or status.",
		Example: `  icctl connectors list-by-location loc_01HW3XYZ
  icctl connectors list-by-location loc_01HW3XYZ -o json
  icctl connectors list-by-location loc_01HW3XYZ --page-size 50 --max-items 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectorsListByLocation(cmd, args[0])
		},
	}
	cmd.Flags().Int("page-size", 0, "Number of items per page (0 = server default)")
	cmd.Flags().Int("max-items", 0, "Maximum total items to return (0 = unlimited)")
	return cmd
}

func runConnectorsListByLocation(cmd *cobra.Command, locationID string) error {
	state := StateFrom(cmd.Context())
	if state == nil {
		return fmt.Errorf("internal error: CLI state not resolved")
	}

	client, err := tui.BuildClient(state.EffectiveBaseURL, state.EffectiveToken, state.LogLevel)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if state.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, state.Timeout)
		defer cancel()
	}

	var listOpts []intelligencecloud.ListOption
	if ps, _ := cmd.Flags().GetInt("page-size"); ps > 0 {
		listOpts = append(listOpts, intelligencecloud.WithPageSize(ps))
	}
	if mi, _ := cmd.Flags().GetInt("max-items"); mi > 0 {
		listOpts = append(listOpts, intelligencecloud.WithMaxItems(mi))
	}

	iter := client.Connectors.ListByLocation(ctx, locationID, listOpts...)
	defer iter.Close()

	var connectors []intelligencecloud.LocationConnector
	for iter.Next(ctx) {
		connectors = append(connectors, iter.Value())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	pi := iter.PageInfo()
	return formatLocationConnectorsList(cmd, state.EffectiveOutput, connectors, pi)
}

func formatLocationConnectorsList(cmd *cobra.Command, format string, connectors []intelligencecloud.LocationConnector, pi intelligencecloud.PageInfo) error {
	switch format {
	case "json":
		items := make([]interface{}, 0, len(connectors))
		for _, c := range connectors {
			items = append(items, c)
		}
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteList(items, output.PageInfo{
			ItemsFetched:  pi.ItemsFetched,
			PagesFetched:  pi.PagesFetched,
			HasNextPage:   pi.HasNextPage,
			NextPageToken: pi.NextPageToken,
		})
	default:
		headers := []string{"ID", "NAME", "TYPE", "STATUS", "LAST SYNCED"}
		rows := make([][]string, 0, len(connectors))
		for _, c := range connectors {
			lastSynced := "never"
			if c.LastSyncedAt != nil {
				lastSynced = c.LastSyncedAt.Format(time.RFC3339)
			}
			rows = append(rows, []string{
				c.Id,
				c.Name,
				c.TypeDisplayName,
				string(c.StatusDisplay),
				lastSynced,
			})
		}
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		return f.WriteTable(headers, rows)
	}
}

func newConnectorsListByCompanyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-company <company-id>",
		Short: "List connectors for a company",
		Long:  "List connectors scoped to a specific company. Results include both paginated connector data and an aggregate error summary for connectors in error state.",
		Example: `  icctl connectors list-by-company cmp_01HW3XYZ
  icctl connectors list-by-company cmp_01HW3XYZ -o json
  icctl connectors list-by-company cmp_01HW3XYZ --page-size 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectorsListByCompany(cmd, args[0])
		},
	}
	cmd.Flags().Int("page-size", 0, "Number of items per page (0 = server default)")
	cmd.Flags().Int("max-items", 0, "Maximum total items to return (0 = unlimited)")
	return cmd
}

func runConnectorsListByCompany(cmd *cobra.Command, companyID string) error {
	state := StateFrom(cmd.Context())
	if state == nil {
		return fmt.Errorf("internal error: CLI state not resolved")
	}

	client, err := tui.BuildClient(state.EffectiveBaseURL, state.EffectiveToken, state.LogLevel)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if state.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, state.Timeout)
		defer cancel()
	}

	var listOpts []intelligencecloud.ListOption
	if ps, _ := cmd.Flags().GetInt("page-size"); ps > 0 {
		listOpts = append(listOpts, intelligencecloud.WithPageSize(ps))
	}
	if mi, _ := cmd.Flags().GetInt("max-items"); mi > 0 {
		listOpts = append(listOpts, intelligencecloud.WithMaxItems(mi))
	}

	result := client.Connectors.ListByCompany(ctx, companyID, listOpts...)
	defer result.Iterator.Close()

	var connectors []intelligencecloud.CompanyConnector
	for result.Iterator.Next(ctx) {
		connectors = append(connectors, result.Iterator.Value())
	}
	if err := result.Iterator.Err(); err != nil {
		return err
	}

	pi := result.Iterator.PageInfo()
	return formatCompanyConnectorsList(cmd, state.EffectiveOutput, connectors, result.ErrorSummary, pi)
}

func formatCompanyConnectorsList(cmd *cobra.Command, format string, connectors []intelligencecloud.CompanyConnector, errorSummary []intelligencecloud.ConnectorErrorSummaryItem, pi intelligencecloud.PageInfo) error {
	switch format {
	case "json":
		type jsonResponse struct {
			Items        []interface{}          `json:"items"`
			PageInfo     output.PageInfo        `json:"page_info"`
			ErrorSummary []errorSummaryItemJSON `json:"error_summary"`
		}
		items := make([]interface{}, 0, len(connectors))
		for _, c := range connectors {
			items = append(items, c)
		}
		errs := make([]errorSummaryItemJSON, 0, len(errorSummary))
		for _, e := range errorSummary {
			errs = append(errs, errorSummaryItemJSON{
				Name:          e.Name,
				StatusMessage: e.StatusMessage,
			})
		}
		resp := jsonResponse{
			Items: items,
			PageInfo: output.PageInfo{
				ItemsFetched:  pi.ItemsFetched,
				PagesFetched:  pi.PagesFetched,
				HasNextPage:   pi.HasNextPage,
				NextPageToken: pi.NextPageToken,
			},
			ErrorSummary: errs,
		}
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(resp)
	default:
		headers := []string{"ID", "NAME", "LOCATION", "TYPE", "STATUS", "LAST SYNCED"}
		rows := make([][]string, 0, len(connectors))
		for _, c := range connectors {
			lastSynced := "never"
			if c.LastSyncedAt != nil {
				lastSynced = c.LastSyncedAt.Format(time.RFC3339)
			}
			rows = append(rows, []string{
				c.Id,
				c.Name,
				c.LocationName,
				c.TypeDisplayName,
				string(c.StatusDisplay),
				lastSynced,
			})
		}
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		if err := f.WriteTable(headers, rows); err != nil {
			return err
		}

		// Print error summary if any connectors are in error state.
		if len(errorSummary) > 0 {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "Connectors in error state (%d):\n", len(errorSummary))
			errHeaders := []string{"NAME", "STATUS MESSAGE"}
			errRows := make([][]string, 0, len(errorSummary))
			for _, e := range errorSummary {
				errRows = append(errRows, []string{e.Name, e.StatusMessage})
			}
			f2 := output.NewTableFormatter(cmd.OutOrStdout(), true)
			return f2.WriteTable(errHeaders, errRows)
		}

		return nil
	}
}

// errorSummaryItemJSON is the JSON representation of a connector error summary.
type errorSummaryItemJSON struct {
	Name          string `json:"name"`
	StatusMessage string `json:"status_message"`
}
