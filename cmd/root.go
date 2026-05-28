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
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

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
		configCommand(),
		docsCommand(),
	)

	return cmd
}
