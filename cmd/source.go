package cmd

import "github.com/spf13/cobra"

func sourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Source repository commands",
	}
	cmd.AddCommand(sourceListCommand())
	return cmd
}
