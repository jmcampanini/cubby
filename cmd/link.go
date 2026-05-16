package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/spf13/cobra"
)

func linkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Create symlinks for selected profiles",
	}
	addProfileFlag(cmd)
	cmd.Flags().Bool("dry-run", false, "preview planned link actions without modifying files")
	cmd.Flags().Bool("json", false, "print link plan as JSON")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		project, profiles, err := loadProfileScopedProject(cmd)
		if err != nil {
			return err
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		jsonOutput, err := jsonOutputEnabled(cmd)
		if err != nil {
			return err
		}
		return linkProfilesWithOptions(cmd, project, profiles, linkRunOptions{DryRun: dryRun, JSON: jsonOutput})
	}
	return cmd
}

type linkRunOptions struct {
	DryRun bool
	JSON   bool
}

func linkProfilesWithOptions(cmd *cobra.Command, project *config.Project, profiles []string, opts linkRunOptions) error {
	profiles = config.NormalizeProfiles(profiles)
	if err := validateSelectedProfiles(project, profiles); err != nil {
		return err
	}
	if err := renderMissingProfileDiagnostics(cmd, project, profiles); err != nil {
		return err
	}

	discovered, err := discoverProfileFiles(project, profiles)
	if err != nil {
		return err
	}
	plan, err := linkops.PlanLink(project.HostRoot, linkSources(discovered), linkops.PlanOptions{
		IgnoreConflicts: project.Host.IgnoreConflicts,
		CaseSensitive:   project.Host.CaseSensitive,
	})
	if err != nil {
		return err
	}

	out := commandOut(cmd)
	if opts.JSON {
		if plan.HasFatalConflicts() {
			if err := writeCommandJSON(cmd, linkActionsEnvelope(opts.DryRun, plan.Actions)); err != nil {
				return err
			}
			return &ExitError{Code: 1}
		}
		if !opts.DryRun {
			if err := linkops.ApplyLink(plan); err != nil {
				return err
			}
		}
		return writeCommandJSON(cmd, linkActionsEnvelope(opts.DryRun, plan.Actions))
	}

	if opts.DryRun {
		if err := linkops.RenderActions(out, plan.Actions); err != nil {
			return err
		}
		if plan.HasFatalConflicts() {
			return &ExitError{Code: 1}
		}
		return nil
	}

	if plan.HasFatalConflicts() {
		if err := linkops.RenderActions(out, plan.FatalConflicts()); err != nil {
			return err
		}
		return &ExitError{Code: 1}
	}
	if err := linkops.ApplyLink(plan); err != nil {
		return err
	}
	if err := linkops.RenderActions(out, plan.Skips()); err != nil {
		return err
	}
	return nil
}

func linkSources(discovered []discoveredProfileFiles) []linkops.SourceFiles {
	sources := make([]linkops.SourceFiles, 0, len(discovered))
	for _, item := range discovered {
		relPaths := make([]string, 0, len(item.files))
		for _, file := range item.files {
			relPaths = append(relPaths, file.RelPath)
		}
		sources = append(sources, linkops.SourceFiles{
			Name:     item.source.Name,
			Root:     item.source.ResolvedPath,
			RelPaths: relPaths,
		})
	}
	return sources
}

func commandOut(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	return cmd.OutOrStdout()
}

func linkProfileFile(hostRoot, sourceRoot, relPath string) error {
	sourcePath := filepath.Join(sourceRoot, relPath)
	hostPath := filepath.Join(hostRoot, relPath)

	info, err := os.Lstat(hostPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			ok, err := linkops.PointsTo(hostPath, sourcePath)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
			return fmt.Errorf("host path is an unexpected symlink: %s", hostPath)
		}
		return fmt.Errorf("host path already exists: %s", hostPath)
	}
	if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	target, err := linkops.RelativeTarget(hostPath, sourcePath)
	if err != nil {
		return err
	}
	return os.Symlink(target, hostPath)
}
