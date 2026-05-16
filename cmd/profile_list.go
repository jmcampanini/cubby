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
