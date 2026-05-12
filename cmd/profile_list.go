package cmd

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

func profileListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles declared by registered sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}
			for _, profile := range project.DeclaredProfiles() {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), profile); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
