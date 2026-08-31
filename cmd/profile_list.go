package cmd

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

type profileListEnvelope struct {
	Profiles []string `json:"profiles"`
}

func profileListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List profiles declared by registered sources",
		Long: `Print the union of profiles declared by all registered sources, sorted,
one per line on stdout. All sources must load. This is the set every
selected profile must belong to; it does not depend on the current
selection.

` + jsonContractHelp + `
The document is {"profiles": [...]}.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}
			profiles := project.DeclaredProfiles()
			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeCommandJSON(cmd, profileListEnvelope{Profiles: profiles})
			}
			for _, profile := range profiles {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), profile); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "print profile inventory as JSON")
	return cmd
}
