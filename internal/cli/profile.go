package cli

import "github.com/spf13/cobra"

func profileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile commands",
		Long: `Inspect profiles. A profile is a name a source declares in cubby.toml; a
source file whose basename carries .<profile> belongs to that profile, and
the host selects which profiles to link.

` + commandGroupHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(profileListCommand(), profileEffectiveCommand())
	return cmd
}
