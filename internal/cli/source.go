package cli

import "github.com/spf13/cobra"

func sourceCommand(streams commandIO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Source repository commands",
		Long: `Inspect registered sources. A source is a directory registered by a
[[source]] entry in the host .cubby.toml (a name plus a path that is
absolute, ~/..., or relative to the host root) that contains a cubby.toml
declaring at least one profile. 'list' prints the registered sources; run
without a subcommand this prints help.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(sourceListCommand(streams))
	streams.bind(cmd)
	return cmd
}
