package cmd

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

func gitignoreCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Report missing required .gitignore patterns",
		Long:  "Load the host and source Cubby configs, compute required patterns for all declared profiles, and print each missing .gitignore pattern. Exits non-zero when any pattern is missing.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}

			missing, err := missingPatterns(project)
			if err != nil {
				return err
			}
			for _, pattern := range missing {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), pattern); err != nil {
					return err
				}
			}
			if len(missing) > 0 {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
}
