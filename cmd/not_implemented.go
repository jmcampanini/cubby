package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func profileNotImplementedCommand(use, short string) *cobra.Command {
	cmd := notImplementedCommand(use, short)
	cmd.Flags().StringSlice("profile", nil, "profile name (repeatable or comma-separated)")
	return cmd
}

func sourceNotImplementedCommand(use, short string) *cobra.Command {
	cmd := notImplementedCommand(use, short)
	cmd.Flags().String("source", "", "registered source name")
	return cmd
}

func notImplementedCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s: not implemented", cmd.CommandPath())
		},
	}
}
