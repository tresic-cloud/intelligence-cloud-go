package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/confirm"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/tui"
)

// NOTE: The "delete" subcommand is intentionally omitted. The generated OpenAPI
// client (internal/generated/client.gen.go) does not include a DeleteReseller
// operation. If a hard-delete endpoint is added to the API in the future, a
// corresponding "resellers delete" subcommand should be added here following
// the destructive-verb pattern used by "deactivate".

// NewResellersCmd creates the resellers subcommand tree with all sub-subcommands.
func NewResellersCmd(profileLoader ProfileLoader, secretLoader SecretLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resellers",
		Short: "Manage resellers",
		Long:  "Create, read, update, deactivate, reactivate, and audit reseller accounts.",
		Example: `  # List all resellers
  icctl resellers list

  # Get a specific reseller
  icctl resellers get rsl_01HW3XYZ

  # Create a reseller from a JSON file
  icctl resellers create --from-file reseller.json

  # Deactivate a reseller (requires confirmation)
  icctl resellers deactivate rsl_01HW3XYZ --reason "contract ended"`,
	}

	cmd.AddCommand(
		newResellersListCmd(),
		newResellersGetCmd(),
		newResellersCreateCmd(),
		newResellersUpdateCmd(),
		newResellersPatchCmd(),
		newResellersDeactivateCmd(),
		newResellersReactivateCmd(),
		newResellersAuditLogsCmd(),
	)

	return cmd
}

// ---------------------------------------------------------------------------
// resellers list
// ---------------------------------------------------------------------------

func newResellersListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resellers",
		Long:  "List all resellers with optional pagination and filtering.",
		Example: `  icctl resellers list
  icctl resellers list --page-size 50
  icctl resellers list --filter country=US
  icctl resellers list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersList(cmd)
		},
	}

	cmd.Flags().Int("page-size", 0, "Number of results per page")
	cmd.Flags().Int("max-items", 0, "Maximum total items to return (0 = unlimited)")
	cmd.Flags().StringSlice("filter", nil, "Filter in key=value format (repeatable)")

	return cmd
}

func runResellersList(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	var opts []intelligencecloud.ListOption

	pageSize, _ := cmd.Flags().GetInt("page-size")
	if pageSize > 0 {
		opts = append(opts, intelligencecloud.WithPageSize(pageSize))
	}

	maxItems, _ := cmd.Flags().GetInt("max-items")
	if maxItems > 0 {
		opts = append(opts, intelligencecloud.WithMaxItems(maxItems))
	}

	filters, _ := cmd.Flags().GetStringSlice("filter")
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) == 2 {
			opts = append(opts, intelligencecloud.WithFilter(parts[0], parts[1]))
		}
	}

	iter := client.Resellers.List(ctx, opts...)
	defer iter.Close()

	var items []interface{}
	for iter.Next(ctx) {
		items = append(items, iter.Value())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	pageInfo := iter.PageInfo()

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteList(items, output.PageInfo{
			ItemsFetched:  pageInfo.ItemsFetched,
			PagesFetched:  pageInfo.PagesFetched,
			HasNextPage:   pageInfo.HasNextPage,
			NextPageToken: pageInfo.NextPageToken,
		})
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"ID", "NAME", "ACTIVE", "CREATED"}
		var rows [][]string
		for _, item := range items {
			r, ok := item.(intelligencecloud.Reseller)
			if !ok {
				continue
			}
			rows = append(rows, []string{
				ptrStr(r.Id),
				ptrStr(r.ResellerName),
				output.FormatBool(ptrBool(r.IsActive), true),
				ptrTimeStr(r.CreatedAt),
			})
		}
		return f.WriteTable(headers, rows)
	}
}

// ---------------------------------------------------------------------------
// resellers get
// ---------------------------------------------------------------------------

func newResellersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a reseller by ID",
		Long:  "Retrieve a single reseller by its unique identifier.",
		Example: `  icctl resellers get rsl_01HW3XYZ
  icctl resellers get rsl_01HW3XYZ --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersGet(cmd, args[0])
		},
	}
}

func runResellersGet(cmd *cobra.Command, id string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	r, err := client.Resellers.Get(ctx, id)
	if err != nil {
		return err
	}

	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers create
// ---------------------------------------------------------------------------

func newResellersCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new reseller",
		Long:  "Create a new reseller account. Provide input via --from-file with a JSON file.",
		Example: `  icctl resellers create --from-file reseller.json
  icctl resellers create --from-file reseller.json --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersCreate(cmd)
		},
	}

	cmd.Flags().String("from-file", "", "Path to JSON file with request body")

	return cmd
}

func runResellersCreate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required for resellers create")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.CreateResellerRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
	}

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	r, err := client.Resellers.Create(ctx, req)
	if err != nil {
		return err
	}

	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers update
// ---------------------------------------------------------------------------

func newResellersUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a reseller (full replacement)",
		Long:  "Update all fields of a reseller by ID. Uses PUT semantics (full replacement). Provide input via --from-file.",
		Example: `  icctl resellers update rsl_01HW3XYZ --from-file reseller.json`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersUpdate(cmd, args[0])
		},
	}

	cmd.Flags().String("from-file", "", "Path to JSON file with request body")

	return cmd
}

func runResellersUpdate(cmd *cobra.Command, id string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required for resellers update")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.UpdateResellerRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
	}

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	r, err := client.Resellers.Update(ctx, id, req)
	if err != nil {
		return err
	}

	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers patch
// ---------------------------------------------------------------------------

func newResellersPatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch <id>",
		Short: "Patch a reseller (partial update)",
		Long:  "Partially update a reseller by ID. Uses PATCH semantics (only provided fields are updated). Provide input via --from-file.",
		Example: `  icctl resellers patch rsl_01HW3XYZ --from-file patch.json`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersPatch(cmd, args[0])
		},
	}

	cmd.Flags().String("from-file", "", "Path to JSON file with partial update")

	return cmd
}

func runResellersPatch(cmd *cobra.Command, id string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required for resellers patch")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.PatchResellerRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
	}

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	r, err := client.Resellers.Patch(ctx, id, req)
	if err != nil {
		return err
	}

	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers deactivate (DESTRUCTIVE)
// ---------------------------------------------------------------------------

func newResellersDeactivateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate <id>",
		Short: "Deactivate a reseller",
		Long:  "Deactivate a reseller account. This is a destructive operation that requires confirmation. Use --reason to specify the deactivation reason (required for audit).",
		Example: `  # Interactive confirmation
  icctl resellers deactivate rsl_01HW3XYZ --reason "contract ended"

  # Bypass confirmation
  icctl resellers deactivate rsl_01HW3XYZ --reason "contract ended" --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersDeactivate(cmd, args[0])
		},
	}

	cmd.Flags().String("reason", "", "Reason for deactivation (required)")
	cmd.Flags().Bool("force-tty", false, "Force TTY mode for testing")
	// Hide the test-only flag.
	_ = cmd.Flags().MarkHidden("force-tty")

	return cmd
}

func runResellersDeactivate(cmd *cobra.Command, id string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	reason, _ := cmd.Flags().GetString("reason")
	if reason == "" {
		return fmt.Errorf("--reason is required for resellers deactivate")
	}

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	// Fetch the current entity for the preview.
	current, err := client.Resellers.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch reseller for preview: %w", err)
	}

	attrs := map[string]string{
		"name":   ptrStr(current.ResellerName),
		"status": resellerStatus(current),
	}

	preview := confirm.DestructivePreview{
		Environment: confirm.EnvironmentInfo{
			Name:    state.EffectiveEnvironment,
			BaseURL: state.EffectiveBaseURL,
		},
		Operation:    "Deactivate reseller",
		ResourceType: "reseller",
		ResourceID:   id,
		Attributes:   attrs,
		Irreversible: false, // reactivate exists
	}

	// Emit preview based on output format.
	switch state.EffectiveOutput {
	case "json":
		if err := confirm.RenderJSONPreview(cmd.OutOrStdout(), preview); err != nil {
			return err
		}
	default:
		if err := confirm.RenderTablePreview(cmd.ErrOrStderr(), preview); err != nil {
			return err
		}
	}

	// Determine TTY status: test mode via --force-tty, otherwise non-TTY (in-process tests).
	forceTTY, _ := cmd.Flags().GetBool("force-tty")
	isTTY := forceTTY

	// Prompt for confirmation.
	if err := confirm.Prompt(confirm.PromptOptions{
		AssumeYes: state.AssumeYes,
		IsTTY:     isTTY,
		Stdin:     cmd.InOrStdin(),
		Stderr:    cmd.ErrOrStderr(),
	}); err != nil {
		return err
	}

	// Execute the deactivation.
	req := intelligencecloud.DeactivateResellerRequest{
		Reason: reason,
	}

	r, err := client.Resellers.Deactivate(ctx, id, req)
	if err != nil {
		return err
	}

	// For JSON output, emit the result after the plan.
	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers reactivate (NOT destructive)
// ---------------------------------------------------------------------------

func newResellersReactivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reactivate <id>",
		Short: "Reactivate a reseller",
		Long:  "Reactivate a previously deactivated reseller. This is not a destructive operation and does not require confirmation.",
		Example: `  icctl resellers reactivate rsl_01HW3XYZ`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersReactivate(cmd, args[0])
		},
	}
}

func runResellersReactivate(cmd *cobra.Command, id string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	r, err := client.Resellers.Reactivate(ctx, id)
	if err != nil {
		return err
	}

	return formatResellerEntity(cmd, state, r)
}

// ---------------------------------------------------------------------------
// resellers audit-logs
// ---------------------------------------------------------------------------

func newResellersAuditLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-logs <id>",
		Short: "List audit logs for a reseller",
		Long:  "List audit log entries for a specific reseller with optional filtering by action, date range, actor, and search text.",
		Example: `  icctl resellers audit-logs rsl_01HW3XYZ
  icctl resellers audit-logs rsl_01HW3XYZ --since 2026-01-01 --until 2026-04-01
  icctl resellers audit-logs rsl_01HW3XYZ --action reseller.updated`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResellersAuditLogs(cmd, args[0])
		},
	}

	cmd.Flags().String("action", "", "Filter by action type")
	cmd.Flags().String("since", "", "Filter by performed_at >= date (ISO 8601)")
	cmd.Flags().String("until", "", "Filter by performed_at <= date (ISO 8601)")
	cmd.Flags().String("actor-id", "", "Filter by actor ID")
	cmd.Flags().String("search", "", "Search in audit log details")

	return cmd
}

func runResellersAuditLogs(cmd *cobra.Command, resellerID string) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildResellerClient(state)
	if err != nil {
		return err
	}

	var opts []intelligencecloud.ListOption

	if v, _ := cmd.Flags().GetString("action"); v != "" {
		opts = append(opts, intelligencecloud.WithFilter("action", v))
	}
	if v, _ := cmd.Flags().GetString("since"); v != "" {
		opts = append(opts, intelligencecloud.WithFilter("start_date", v))
	}
	if v, _ := cmd.Flags().GetString("until"); v != "" {
		opts = append(opts, intelligencecloud.WithFilter("end_date", v))
	}
	if v, _ := cmd.Flags().GetString("actor-id"); v != "" {
		opts = append(opts, intelligencecloud.WithFilter("actor_id", v))
	}
	if v, _ := cmd.Flags().GetString("search"); v != "" {
		opts = append(opts, intelligencecloud.WithFilter("search", v))
	}

	iter := client.Resellers.ListAuditLogs(ctx, resellerID, opts...)
	defer iter.Close()

	var items []interface{}
	for iter.Next(ctx) {
		items = append(items, iter.Value())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	pageInfo := iter.PageInfo()

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteList(items, output.PageInfo{
			ItemsFetched:  pageInfo.ItemsFetched,
			PagesFetched:  pageInfo.PagesFetched,
			HasNextPage:   pageInfo.HasNextPage,
			NextPageToken: pageInfo.NextPageToken,
		})
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"ID", "ACTION", "ACTOR", "DETAILS", "TIMESTAMP"}
		var rows [][]string
		for _, item := range items {
			entry, ok := item.(intelligencecloud.AuditLogEntry)
			if !ok {
				continue
			}
			rows = append(rows, []string{
				entry.Id,
				entry.Action,
				entry.ActorName,
				entry.Details,
				entry.Timestamp.Format("2006-01-02 15:04:05"),
			})
		}
		return f.WriteTable(headers, rows)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildResellerClient constructs an SDK client from the CLI state.
func buildResellerClient(state *State) (*intelligencecloud.Client, error) {
	if state == nil {
		return nil, fmt.Errorf("CLI state not available")
	}
	return tui.BuildClient(state.EffectiveBaseURL, state.EffectiveToken, state.LogLevel)
}

// formatResellerEntity formats a single reseller for output.
func formatResellerEntity(cmd *cobra.Command, state *State, r *intelligencecloud.Reseller) error {
	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(r)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", ptrStr(r.Id)},
			{"Name", ptrStr(r.ResellerName)},
			{"Active", output.FormatBool(ptrBool(r.IsActive), true)},
			{"Contact Name", ptrStr(r.PrimaryContactName)},
			{"Contact Email", ptrStr(r.PrimaryContactEmail)},
			{"Contact Phone", ptrStr(r.PrimaryContactPhone)},
			{"Country", ptrStr(r.Country)},
			{"Created", ptrTimeStr(r.CreatedAt)},
			{"Updated", ptrTimeStr(r.UpdatedAt)},
		}
		return f.WriteTable(headers, rows)
	}
}

// ptrStr safely dereferences a *string, returning "" if nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrBool safely dereferences a *bool, returning false if nil.
func ptrBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// ptrTimeStr safely formats a *time.Time, returning "" if nil.
func ptrTimeStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// resellerStatus returns a human-readable status string.
func resellerStatus(r *intelligencecloud.Reseller) string {
	if ptrBool(r.IsActive) {
		return "active"
	}
	return "inactive"
}
