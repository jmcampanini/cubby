package cli

import "github.com/spf13/cobra"

func gitignoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitignore",
		Short: "Check or update required host .gitignore patterns",
		Long: `Keep the host .gitignore covering .cubby.toml and profile-scoped files.

` + commandGroupHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(gitignoreCheckCommand(), gitignoreSyncCommand())
	return cmd
}
