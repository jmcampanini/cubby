package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/spf13/cobra"
)

type pruneEnvelope struct {
	Removed []pruneRemoved `json:"removed"`
}

type pruneRemoved struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func pruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove dangling Cubby symlinks",
		Long: `Remove every managed link whose target no longer exists and print each
removed host path on stdout, one per line. Exit status is 0 whether or not
anything was removed. Links whose target exists are left alone whatever
other drift they show, regular files and unmanaged symlinks are never
touched, and empty parent directories are not removed. Removal stops at
the first filesystem error.

A registered source whose directory is missing is still considered, so
links into a source that was deleted or moved are pruned. A source entry
without a path, a path that is not a directory, or an invalid cubby.toml
is an error and nothing is removed.

` + managedLinkHelp + `

` + jsonContractHelp + `
The document is {"removed": [{"path", "source", "target"}]}.`,
		Args: cobra.NoArgs,
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
			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			removed := make([]pruneRemoved, 0)
			for _, link := range links {
				if link.TargetExists {
					continue
				}
				if err := os.Remove(link.HostPath); err != nil {
					return err
				}
				item := pruneRemoved{
					Path:   slashPath(link.HostRelPath),
					Source: link.SourceName,
					Target: slashPath(link.SourceRelPath),
				}
				removed = append(removed, item)
				if !jsonOutput {
					if _, err := fmt.Fprintln(commandOut(cmd), filepath.ToSlash(link.HostRelPath)); err != nil {
						return err
					}
				}
			}
			if jsonOutput {
				return writeCommandJSON(cmd, pruneEnvelope{Removed: removed})
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "print prune results as JSON")
	return cmd
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
