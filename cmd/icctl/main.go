// Package main is the entry point for the icctl CLI binary. It sets up
// SIGINT handling via signal.NotifyContext so that in-flight API calls are
// cancelled when the user presses Ctrl-C, and maps the resulting context
// cancellation to exit code 130 per cli-schema.md.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/cmd"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Set up SIGINT (Ctrl-C) handling. When a signal is received, ctx is
	// cancelled, which propagates to in-flight SDK calls.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Build the root command. profileLoader and secretLoader are nil here;
	// they will be wired to the real implementations from W3-1's profile
	// package at integration time.
	rootCmd := cmd.NewRootCmd(nil, nil)
	rootCmd.SetContext(ctx)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// Check if the error is due to SIGINT / context cancellation.
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "Interrupted.")
			return cmd.ExitInterrupted
		}

		// Map errors to exit codes per cli-schema.md.
		exitCode := cmd.MapErrorToExitCode(err)
		if exitCode != 0 {
			fmt.Fprintln(os.Stderr, err)
		}
		return exitCode
	}

	return 0
}
