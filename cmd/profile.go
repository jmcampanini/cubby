package cmd

import "github.com/spf13/cobra"

func profileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile commands",
	}
	cmd.AddCommand(profileListCommand())
	return cmd
}
