package cmd

import "github.com/spf13/cobra"

func lazygitCommand() *cobra.Command {
	return sourceNotImplementedCommand("lazygit", "Open lazygit in a source repo")
}
