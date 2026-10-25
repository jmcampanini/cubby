package cli

import (
	"github.com/spf13/cobra"
)

func gitignoreCommand(streams commandIO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitignore",
		Short: "Check or update required host .gitignore patterns",
		Long: `Keep the host .gitignore covering .cubby.toml and profile-scoped files.
'check' reports the required patterns that are missing and 'sync' appends
them; run without a subcommand this prints help.

` + gitignorePatternsHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(gitignoreCheckCommand(streams), gitignoreSyncCommand(streams))
	streams.bind(cmd)
	return cmd
}
