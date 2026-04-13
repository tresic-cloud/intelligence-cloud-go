// Package cmd defines the cobra command tree for icctl.
package cmd

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd creates a new completion subcommand. Each call returns a
// fresh *cobra.Command so the function is safe to use from concurrent tests
// and from multiple root commands.
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for icctl.

The generated script enables tab-completion for all icctl commands, flags,
and arguments in the specified shell. Load the script according to the
instructions for your shell below.`,
		Example: `  # Bash — load in current session
  source <(icctl completion bash)

  # Bash — install permanently
  icctl completion bash > /etc/bash_completion.d/icctl

  # Zsh — load in current session
  source <(icctl completion zsh)

  # Zsh — install permanently (one-time)
  icctl completion zsh > "${fpath[1]}/_icctl"

  # Fish — load in current session
  icctl completion fish | source

  # Fish — install permanently
  icctl completion fish > ~/.config/fish/completions/icctl.fish

  # PowerShell — load in current session
  icctl completion powershell | Out-String | Invoke-Expression

  # PowerShell — install permanently
  icctl completion powershell >> $PROFILE`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				// This branch is unreachable because cobra.OnlyValidArgs
				// rejects unknown values before RunE is called.
				return nil
			}
		},
		// SilenceUsage prevents cobra from printing the usage message on error
		// from RunE; argument validation errors still show usage via cobra's
		// built-in mechanism.
		SilenceUsage: true,
	}
}

// RegisterCompletionCmd adds the completion subcommand to the given root command.
// Call this from the root command setup (e.g. in root.go's init or Execute function).
func RegisterCompletionCmd(root *cobra.Command) {
	root.AddCommand(newCompletionCmd())
}
