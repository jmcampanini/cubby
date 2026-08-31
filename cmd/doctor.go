package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/spf13/cobra"
)

type doctorEnvelope struct {
	Healthy bool          `json:"healthy"`
	Issues  []doctorIssue `json:"issues"`
}

type doctorIssue struct {
	Kind    string   `json:"kind"`
	Source  string   `json:"source,omitempty"`
	Message string   `json:"message,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Profile string   `json:"profile,omitempty"`
	Path    string   `json:"path,omitempty"`
	Target  string   `json:"target,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

func doctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run Cubby health checks",
		Long: `Run health checks against the host and its registered sources, print one
line per issue on stdout, and exit 1 when any issue is found. A healthy
setup prints nothing and exits 0. Nothing is modified.

Checks and their stdout markers:
  MISSING_SOURCE     a registered source path is missing, is not a
                     directory, or has an invalid cubby.toml; the other
                     checks continue with the remaining sources
  MISSING_GITIGNORE  a pattern 'cubby gitignore check' would report
  MISSING_PROFILE    a selected profile that no loadable source declares
  DANGLING           a managed link whose target is missing
  DRIFT              a managed link with any other drift reason
  CONFLICT           a host path 'cubby link' would refuse for the
                     selected profiles; reported even when
                     ignore_conflicts or --ignore-conflicts is set

Profiles are selected as for 'cubby link'; with none selected the CONFLICT
check is skipped rather than failing. --case-sensitive changes
case-collision detection as it does for link. Links into a source whose
directory is missing are still found and reported as DANGLING.

` + profileSelectionHelp + `

` + managedLinkHelp + `

` + driftReasonsHelp + `

` + jsonContractHelp + `
The document is {"healthy": bool, "issues": [...]}, each issue carrying
"kind" (missing_source, missing_gitignore, missing_profile, dangling,
drift, or conflict) and the fields shown on its text line.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			hostRoot, hostCfg, err := loadEffectiveHostConfig(cmd)
			if err != nil {
				return err
			}
			project, err := config.LoadProjectDiagnosticsWithHostConfig(hostRoot, hostCfg)
			if err != nil {
				return err
			}
			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			issues, err := collectDoctorIssues(project)
			if err != nil {
				return err
			}
			if jsonOutput {
				if err := writeCommandJSON(cmd, doctorEnvelope{Healthy: len(issues) == 0, Issues: issues}); err != nil {
					return err
				}
			} else if err := renderDoctorIssues(cmd, issues); err != nil {
				return err
			}
			if len(issues) > 0 {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
	addProfileFlag(cmd)
	cmd.Flags().Bool("json", false, "print health checks as JSON")
	return cmd
}

func collectDoctorIssues(diagnostic *config.DiagnosticProject) ([]doctorIssue, error) {
	issues := make([]doctorIssue, 0)
	project := diagnostic.Project()

	for _, issue := range diagnostic.SourceIssues {
		issues = append(issues, doctorIssue{
			Kind:    "missing_source",
			Source:  issue.Name,
			Message: issue.Error(),
		})
	}

	missing, err := missingPatterns(project)
	if err != nil {
		return nil, err
	}
	for _, pattern := range missing {
		issues = append(issues, doctorIssue{Kind: "missing_gitignore", Pattern: pattern})
	}

	declared := declaredProfileSet(project)
	requested := config.EffectiveProfiles(diagnostic.Host)
	validRequested := make([]string, 0, len(requested))
	for _, profile := range requested {
		if _, ok := declared[profile]; !ok {
			issues = append(issues, doctorIssue{Kind: "missing_profile", Profile: profile})
			continue
		}
		validRequested = append(validRequested, profile)
	}

	links, err := hostlinks.DiscoverDiagnostics(diagnostic.HostRoot, diagnostic.Sources, diagnostic.SourceRoots)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		if !link.TargetExists {
			issues = append(issues, doctorIssue{
				Kind:   "dangling",
				Path:   slashPath(link.HostRelPath),
				Source: link.SourceName,
				Target: slashPath(link.SourceRelPath),
			})
		}
		drift := nonDanglingReasons(link.DriftReasons)
		if len(drift) > 0 {
			issues = append(issues, doctorIssue{
				Kind:    "drift",
				Path:    slashPath(link.HostRelPath),
				Source:  link.SourceName,
				Target:  slashPath(link.SourceRelPath),
				Reasons: drift,
			})
		}
	}

	if len(validRequested) > 0 {
		discovered, err := discoverProfileFiles(project, validRequested)
		if err != nil {
			return nil, err
		}
		plan, err := linkops.PlanLink(diagnostic.HostRoot, linkSources(discovered), linkops.PlanOptions{
			IgnoreConflicts: false,
			CaseSensitive:   diagnostic.Host.CaseSensitive,
		})
		if err != nil {
			return nil, err
		}
		conflicts := plan.FatalConflicts()
		sort.Slice(conflicts, func(i, j int) bool {
			if conflicts[i].RelPath == conflicts[j].RelPath {
				return conflicts[i].SourceName < conflicts[j].SourceName
			}
			return conflicts[i].RelPath < conflicts[j].RelPath
		})
		for _, conflict := range conflicts {
			issues = append(issues, doctorIssue{
				Kind:   "conflict",
				Path:   slashPath(conflict.RelPath),
				Source: conflict.SourceName,
				Reason: conflict.Reason,
			})
		}
	}

	return issues, nil
}

func renderDoctorIssues(cmd *cobra.Command, issues []doctorIssue) error {
	out := commandOut(cmd)
	for _, issue := range issues {
		switch issue.Kind {
		case "missing_source":
			if _, err := fmt.Fprintf(out, "MISSING_SOURCE %s %s\n", issue.Source, issue.Message); err != nil {
				return err
			}
		case "missing_gitignore":
			if _, err := fmt.Fprintf(out, "MISSING_GITIGNORE %s\n", issue.Pattern); err != nil {
				return err
			}
		case "missing_profile":
			if _, err := fmt.Fprintf(out, "MISSING_PROFILE %s\n", issue.Profile); err != nil {
				return err
			}
		case "dangling":
			if _, err := fmt.Fprintf(out, "DANGLING %s [source=%s target=%s]\n", issue.Path, issue.Source, issue.Target); err != nil {
				return err
			}
		case "drift":
			if _, err := fmt.Fprintf(out, "DRIFT %s [source=%s target=%s reasons=%s]\n", issue.Path, issue.Source, issue.Target, strings.Join(issue.Reasons, ",")); err != nil {
				return err
			}
		case "conflict":
			if _, err := fmt.Fprintf(out, "CONFLICT %s %s [source=%s]\n", issue.Path, issue.Reason, issue.Source); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(out, "%s %s\n", strings.ToUpper(issue.Kind), issue.Message); err != nil {
				return err
			}
		}
	}
	return nil
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
