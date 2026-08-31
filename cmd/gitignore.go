package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/gitignore"
	"github.com/spf13/cobra"
)

func gitignoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitignore",
		Short: "Check or update required host .gitignore patterns",
		Long: `Keep the host .gitignore covering .cubby.toml and profile-scoped files.
'check' reports the required patterns that are missing and 'sync' appends
them; run without a subcommand this prints help.

` + gitignorePatternsHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(gitignoreCheckCommand(), gitignoreSyncCommand())
	return cmd
}

func missingPatterns(project *config.Project) ([]string, error) {
	profilePatterns := gitignore.RequiredPatterns(project.DeclaredProfiles())
	required := append([]string{"/" + config.HostConfigFileName}, profilePatterns...)
	gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
	missing, err := gitignore.MissingPatternsFile(gitignorePath, required)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", gitignorePath, err)
	}
	return missing, nil
}
