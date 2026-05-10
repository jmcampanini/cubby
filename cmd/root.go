package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the cubby command tree.
func NewRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cubby",
		Short:         "Layer profile-scoped dotfiles into a host repo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.AddCommand(
		linkCommand(),
		unlinkCommand(),
		pruneCommand(),
		statusCommand(),
		doctorCommand(),
		profileCommand(),
		sourceCommand(),
		gitignoreCommand(),
		lazygitCommand(),
	)

	return cmd
}
