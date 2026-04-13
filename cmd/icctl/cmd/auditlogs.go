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

// NewAuditLogsCmd creates the audit-logs command tree with the list subcommand.
func NewAuditLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-logs",
		Short: "Access audit log entries",
		Long:  "Access audit log entries in the Intelligence Cloud platform. Currently supports listing audit logs for a specific reseller.",
		Example: `  # List audit logs for a reseller
  icctl audit-logs list --reseller rsl_01HW3XYZ

  # Filter by action and date range
  icctl audit-logs list --reseller rsl_01HW3XYZ --action user.login --since 2026-01-01 --until 2026-04-01`,
	}

	cmd.AddCommand(newAuditLogsListCmd())

	return cmd
}

func newAuditLogsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		Long: `List audit log entries for a reseller. Results are paginated and support
filtering by action type, date range, actor ID, and search terms.`,
		Example: `  icctl audit-logs list --reseller rsl_01HW3XYZ
  icctl audit-logs list --reseller rsl_01HW3XYZ --action user.login
  icctl audit-logs list --reseller rsl_01HW3XYZ --since 2026-01-01 --until 2026-04-01
  icctl audit-logs list --reseller rsl_01HW3XYZ --page-size 50 --max-items 200 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuditLogsList(cmd)
		},
	}
	cmd.Flags().String("reseller", "", "Reseller ID (required)")
	_ = cmd.MarkFlagRequired("reseller")
	cmd.Flags().String("action", "", "Filter by action type")
	cmd.Flags().String("since", "", "Filter entries from this date (YYYY-MM-DD)")
	cmd.Flags().String("until", "", "Filter entries until this date (YYYY-MM-DD)")
	cmd.Flags().String("actor-id", "", "Filter by actor ID")
	cmd.Flags().String("search", "", "Search term")
	cmd.Flags().Int("page-size", 0, "Number of items per page (0 = server default)")
	cmd.Flags().Int("max-items", 0, "Maximum total items to return (0 = unlimited)")
	return cmd
}

func runAuditLogsList(cmd *cobra.Command) error {
	state := StateFrom(cmd.Context())
	if state == nil {
		return fmt.Errorf("internal error: CLI state not resolved")
	}

	resellerID, _ := cmd.Flags().GetString("reseller")
	if resellerID == "" {
		return fmt.Errorf("--reseller is required")
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
	if action, _ := cmd.Flags().GetString("action"); action != "" {
		listOpts = append(listOpts, intelligencecloud.WithFilter("action", action))
	}
	if since, _ := cmd.Flags().GetString("since"); since != "" {
		listOpts = append(listOpts, intelligencecloud.WithFilter("start_date", since))
	}
	if until, _ := cmd.Flags().GetString("until"); until != "" {
		listOpts = append(listOpts, intelligencecloud.WithFilter("end_date", until))
	}
	if actorID, _ := cmd.Flags().GetString("actor-id"); actorID != "" {
		listOpts = append(listOpts, intelligencecloud.WithFilter("actor_id", actorID))
	}
	if search, _ := cmd.Flags().GetString("search"); search != "" {
		listOpts = append(listOpts, intelligencecloud.WithFilter("search", search))
	}

	iter := client.AuditLogs.ListForReseller(ctx, resellerID, listOpts...)
	defer iter.Close()

	var entries []intelligencecloud.AuditLogEntry
	for iter.Next(ctx) {
		entries = append(entries, iter.Value())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	pi := iter.PageInfo()
	return formatAuditLogsList(cmd, state.EffectiveOutput, entries, pi)
}

func formatAuditLogsList(cmd *cobra.Command, format string, entries []intelligencecloud.AuditLogEntry, pi intelligencecloud.PageInfo) error {
	switch format {
	case "json":
		items := make([]interface{}, 0, len(entries))
		for _, e := range entries {
			items = append(items, e)
		}
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteList(items, output.PageInfo{
			ItemsFetched:  pi.ItemsFetched,
			PagesFetched:  pi.PagesFetched,
			HasNextPage:   pi.HasNextPage,
			NextPageToken: pi.NextPageToken,
		})
	default:
		headers := []string{"ID", "TIMESTAMP", "ACTION", "ACTOR", "DETAILS"}
		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{
				e.Id,
				e.Timestamp.Format(time.RFC3339),
				e.Action,
				e.ActorName,
				e.Details,
			})
		}
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		return f.WriteTable(headers, rows)
	}
}
