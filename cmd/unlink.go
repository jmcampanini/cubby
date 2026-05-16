package cmd

import (
	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/spf13/cobra"
)

func unlinkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove symlinks for selected profiles",
	}
	addProfileFlag(cmd)
	cmd.Flags().Bool("dry-run", false, "preview planned unlink actions without modifying files")
	cmd.Flags().Bool("json", false, "print unlink plan as JSON")
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
		return unlinkProfilesWithOptions(cmd, project, profiles, unlinkRunOptions{DryRun: dryRun, JSON: jsonOutput})
	}
	return cmd
}

type unlinkRunOptions struct {
	DryRun bool
	JSON   bool
}

func unlinkProfilesWithOptions(cmd *cobra.Command, project *config.Project, profiles []string, opts unlinkRunOptions) error {
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
	plan, err := linkops.PlanUnlink(project.HostRoot, linkSources(discovered))
	if err != nil {
		return err
	}
	if opts.JSON {
		if !opts.DryRun {
			if err := linkops.ApplyUnlink(plan); err != nil {
				return err
			}
		}
		return writeCommandJSON(cmd, linkActionsEnvelope(opts.DryRun, plan.Actions))
	}
	if opts.DryRun {
		return linkops.RenderActions(commandOut(cmd), plan.Actions)
	}
	return linkops.ApplyUnlink(plan)
}
