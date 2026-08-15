package cmd

import "github.com/spf13/cobra"

func sourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Source repository commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(sourceListCommand())
	return cmd
}
