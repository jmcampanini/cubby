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
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		project, profiles, err := loadProfileScopedProject(cmd)
		if err != nil {
			return err
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		return unlinkProfilesWithOptions(cmd, project, profiles, unlinkRunOptions{DryRun: dryRun})
	}
	return cmd
}

type unlinkRunOptions struct {
	DryRun bool
}

func unlinkProfilesWithOptions(cmd *cobra.Command, project *config.Project, profiles []string, opts unlinkRunOptions) error {
	profiles = config.NormalizeProfiles(profiles)
	if err := validateSelectedProfiles(project, profiles); err != nil {
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
	if opts.DryRun {
		return linkops.RenderActions(commandOut(cmd), plan.Actions)
	}
	return linkops.ApplyUnlink(plan)
}
