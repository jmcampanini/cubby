package cmd

import "github.com/spf13/cobra"

func profileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(profileListCommand(), profileEffectiveCommand())
	return cmd
}
