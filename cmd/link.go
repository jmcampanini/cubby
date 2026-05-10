package cmd

import "github.com/spf13/cobra"

func linkCommand() *cobra.Command {
	return profileNotImplementedCommand("link", "Create symlinks for selected profiles")
}
