// Package completion exposes the `everestctl completion` subcommand so
// users can install shell completions for bash, zsh, fish, and powershell.
package completion

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func NewCommand(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Long: `Generate the autocompletion script for the specified shell.

Examples:
  # bash
  source <(everestctl completion bash)
  everestctl completion bash > /etc/bash_completion.d/everestctl

  # zsh
  everestctl completion zsh > "${fpath[1]}/_everestctl"

  # fish
  everestctl completion fish | source

  # powershell
  everestctl completion powershell | Out-String | Invoke-Expression`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Cobra writes the script to the parent command, so we
			// temporarily retarget its output to honour our writer.
			root := cmd.Root()
			old := root.OutOrStdout()
			root.SetOut(out)
			defer root.SetOut(old)
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			}
			fmt.Fprintln(os.Stderr, "unsupported shell")
			return fmt.Errorf("unsupported shell %q", args[0])
		},
	}
	return cmd
}
