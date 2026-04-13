package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/tui"
)

// NewUsersCmd creates the users command tree with the patch-company
// subcommand.
func NewUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage user resources",
		Long:  "Manage user resources in the Intelligence Cloud platform.",
		Example: `  # Update a user within a company
  icctl users patch-company cmp_01HW3XYZ usr_01HW3ABC --from-file update.json`,
	}

	cmd.AddCommand(newUsersPatchCompanyCmd())

	return cmd
}

func newUsersPatchCompanyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch-company <company-id> <user-id>",
		Short: "Update a user within a company",
		Long: `Update a user within a company by sending a PATCH request. The request
body is read from a JSON file specified by --from-file. Only the fields
present in the JSON file are updated; omitted fields remain unchanged.`,
		Example: `  # Update user with JSON file
  icctl users patch-company cmp_01HW3XYZ usr_01HW3ABC --from-file update.json

  # Update user with JSON output
  icctl users patch-company cmp_01HW3XYZ usr_01HW3ABC --from-file update.json -o json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsersPatchCompany(cmd, args[0], args[1])
		},
	}
	cmd.Flags().String("from-file", "", "Path to JSON file containing the update request body (required)")
	_ = cmd.MarkFlagRequired("from-file")
	return cmd
}

func runUsersPatchCompany(cmd *cobra.Command, companyID, userID string) error {
	state := StateFrom(cmd.Context())
	if state == nil {
		return fmt.Errorf("internal error: CLI state not resolved")
	}

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.UpdateCompanyUserRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
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

	user, err := client.Users.PatchCompany(ctx, companyID, userID, req)
	if err != nil {
		return err
	}

	return formatCompanyUser(cmd, state.EffectiveOutput, user)
}

func formatCompanyUser(cmd *cobra.Command, format string, user *intelligencecloud.CompanyUser) error {
	switch format {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(user)
	default:
		headers := []string{"ID", "EMAIL", "FIRST NAME", "LAST NAME", "ROLE", "STATUS"}
		id := derefStr(user.Id)
		email := derefStr(user.Email)
		firstName := derefStr(user.FirstName)
		lastName := derefStr(user.LastName)
		role := derefStr(user.Role)
		status := derefStr(user.Status)
		rows := [][]string{{id, email, firstName, lastName, role, status}}
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		return f.WriteTable(headers, rows)
	}
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
