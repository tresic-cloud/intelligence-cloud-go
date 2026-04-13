package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/tui"
)

// NewProductsCmd creates the products command tree with the list subcommand.
func NewProductsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "products",
		Short: "Manage product resources",
		Long:  "Manage product resources in the Intelligence Cloud catalogue.",
		Example: `  # List all products
  icctl products list

  # List products as JSON
  icctl products list -o json`,
	}

	cmd.AddCommand(newProductsListCmd())

	return cmd
}

func newProductsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all products",
		Long:  "List all products in the Intelligence Cloud product catalogue. This endpoint returns all products in a single response (no pagination).",
		Example: `  icctl products list
  icctl products list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductsList(cmd)
		},
	}
}

func runProductsList(cmd *cobra.Command) error {
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

	products, err := client.Products.List(ctx)
	if err != nil {
		return err
	}

	return formatProductsList(cmd, state.EffectiveOutput, products)
}

func formatProductsList(cmd *cobra.Command, format string, products []intelligencecloud.Product) error {
	switch format {
	case "json":
		items := make([]interface{}, 0, len(products))
		for _, p := range products {
			items = append(items, p)
		}
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteList(items, output.PageInfo{
			ItemsFetched: len(products),
			PagesFetched: 1,
		})
	default:
		headers := []string{"KEY", "NAME"}
		rows := make([][]string, 0, len(products))
		for _, p := range products {
			rows = append(rows, []string{p.Key, p.Name})
		}
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		return f.WriteTable(headers, rows)
	}
}

