package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/spf13/cobra"
)

func pruneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove dangling Cubby symlinks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProjectDiagnostics()
			if err != nil {
				return err
			}
			if err := validatePruneSourceIssues(project.SourceIssues); err != nil {
				return err
			}
			links, err := hostlinks.DiscoverDiagnostics(project.HostRoot, project.Sources, project.SourceRoots)
			if err != nil {
				return err
			}
			for _, link := range links {
				if link.TargetExists {
					continue
				}
				if err := os.Remove(link.HostPath); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(commandOut(cmd), filepath.ToSlash(link.HostRelPath)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func validatePruneSourceIssues(issues []config.SourceIssue) error {
	for _, issue := range issues {
		if issue.Kind == "missing_path" && issue.Path != "" {
			continue
		}
		return issue.Err
	}
	return nil
}
