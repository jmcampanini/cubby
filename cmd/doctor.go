package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/spf13/cobra"
)

func doctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run Cubby health checks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			hostRoot, hostCfg, err := loadEffectiveHostConfig(cmd)
			if err != nil {
				return err
			}
			project, err := config.LoadProjectDiagnosticsWithHostConfig(hostRoot, hostCfg)
			if err != nil {
				return err
			}
			hasIssues, err := runDoctor(cmd, project)
			if err != nil {
				return err
			}
			if hasIssues {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
	addProfileFlag(cmd)
	return cmd
}

func runDoctor(cmd *cobra.Command, diagnostic *config.DiagnosticProject) (bool, error) {
	hasIssues := false
	out := commandOut(cmd)
	project := diagnostic.Project()

	for _, issue := range diagnostic.SourceIssues {
		hasIssues = true
		if _, err := fmt.Fprintf(out, "MISSING_SOURCE %s %s\n", issue.Name, issue.Error()); err != nil {
			return false, err
		}
	}

	missing, err := missingPatterns(project)
	if err != nil {
		return false, err
	}
	for _, pattern := range missing {
		hasIssues = true
		if _, err := fmt.Fprintf(out, "MISSING_GITIGNORE %s\n", pattern); err != nil {
			return false, err
		}
	}

	declared := declaredProfileSet(project)
	requested := config.NormalizeProfiles(diagnostic.Host.Profiles)
	validRequested := make([]string, 0, len(requested))
	for _, profile := range requested {
		if _, ok := declared[profile]; !ok {
			hasIssues = true
			if _, err := fmt.Fprintf(out, "MISSING_PROFILE %s\n", profile); err != nil {
				return false, err
			}
			continue
		}
		validRequested = append(validRequested, profile)
	}

	links, err := hostlinks.DiscoverDiagnostics(diagnostic.HostRoot, diagnostic.Sources, diagnostic.SourceRoots)
	if err != nil {
		return false, err
	}
	for _, link := range links {
		if !link.TargetExists {
			hasIssues = true
			if _, err := fmt.Fprintf(out, "DANGLING %s [source=%s target=%s]\n", filepath.ToSlash(link.HostRelPath), link.SourceName, filepath.ToSlash(link.SourceRelPath)); err != nil {
				return false, err
			}
		}
		drift := nonDanglingReasons(link.DriftReasons)
		if len(drift) > 0 {
			hasIssues = true
			if _, err := fmt.Fprintf(out, "DRIFT %s [source=%s target=%s reasons=%s]\n", filepath.ToSlash(link.HostRelPath), link.SourceName, filepath.ToSlash(link.SourceRelPath), strings.Join(drift, ",")); err != nil {
				return false, err
			}
		}
	}

	if len(validRequested) > 0 {
		discovered, err := discoverProfileFiles(project, validRequested)
		if err != nil {
			return false, err
		}
		plan, err := linkops.PlanLink(diagnostic.HostRoot, linkSources(discovered), linkops.PlanOptions{
			IgnoreConflicts: diagnostic.Host.IgnoreConflicts,
			CaseSensitive:   diagnostic.Host.CaseSensitive,
		})
		if err != nil {
			return false, err
		}
		conflicts := plan.FatalConflicts()
		sort.Slice(conflicts, func(i, j int) bool {
			if conflicts[i].RelPath == conflicts[j].RelPath {
				return conflicts[i].SourceName < conflicts[j].SourceName
			}
			return conflicts[i].RelPath < conflicts[j].RelPath
		})
		for _, conflict := range conflicts {
			hasIssues = true
			if _, err := fmt.Fprintf(out, "CONFLICT %s %s [source=%s]\n", filepath.ToSlash(conflict.RelPath), conflict.Reason, conflict.SourceName); err != nil {
				return false, err
			}
		}
	}

	return hasIssues, nil
}

func declaredProfileSet(project *config.Project) map[string]struct{} {
	set := make(map[string]struct{})
	for _, profile := range project.DeclaredProfiles() {
		set[profile] = struct{}{}
	}
	return set
}

func nonDanglingReasons(reasons []string) []string {
	filtered := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if reason == hostlinks.ReasonDangling {
			continue
		}
		filtered = append(filtered, reason)
	}
	return filtered
}
