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
		Long:  "Check or update the host repository's .gitignore patterns for every profile declared by registered source repos.",
	}
	cmd.AddCommand(gitignoreCheckCommand(), gitignoreSyncCommand())
	return cmd
}

func missingPatterns(project *config.Project) ([]string, error) {
	profiles := project.DeclaredProfiles()
	required := gitignore.RequiredPatterns(profiles)
	gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
	missing, err := gitignore.MissingPatternsFile(gitignorePath, required)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", gitignorePath, err)
	}
	return missing, nil
}
