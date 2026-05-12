package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		project, profiles, err := loadProfileScopedProject(cmd)
		if err != nil {
			return err
		}
		return unlinkProfiles(project, profiles)
	}
	return cmd
}

func unlinkProfiles(project *config.Project, profiles []string) error {
	profiles = config.NormalizeProfiles(profiles)
	if err := validateSelectedProfiles(project, profiles); err != nil {
		return err
	}

	discovered, err := discoverProfileFiles(project, profiles)
	if err != nil {
		return err
	}
	for _, item := range discovered {
		for _, file := range item.files {
			if err := unlinkProfileFile(project.HostRoot, item.source.ResolvedPath, file.RelPath); err != nil {
				return fmt.Errorf("unlink %s from source %q: %w", file.RelPath, item.source.Name, err)
			}
		}
	}
	return nil
}

func unlinkProfileFile(hostRoot, sourceRoot, relPath string) error {
	sourcePath := filepath.Join(sourceRoot, relPath)
	hostPath := filepath.Join(hostRoot, relPath)

	info, err := os.Lstat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}

	ok, err := linkops.PointsTo(hostPath, sourcePath)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return os.Remove(hostPath)
}
