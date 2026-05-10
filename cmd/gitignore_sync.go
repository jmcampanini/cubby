package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/gitignore"
	"github.com/spf13/cobra"
)

func gitignoreSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Append missing required .gitignore patterns",
		Long:  "Load the host and source Cubby configs, compute required patterns for all declared profiles, and append each missing pattern to the host repository's .gitignore.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}

			missing, err := missingPatterns(project)
			if err != nil {
				return err
			}
			gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
			if err := gitignore.AppendMissing(gitignorePath, missing); err != nil {
				return fmt.Errorf("update %s: %w", gitignorePath, err)
			}
			for _, pattern := range missing {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), pattern); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
