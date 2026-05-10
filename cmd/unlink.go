package cmd

import "github.com/spf13/cobra"

func unlinkCommand() *cobra.Command {
	return profileNotImplementedCommand("unlink", "Remove symlinks for selected profiles")
}
