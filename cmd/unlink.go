package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/jmcampanini/cubby/internal/profilefiles"
	"github.com/spf13/cobra"
)

func unlinkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove symlinks for selected profiles",
	}
	addProfileFlag(cmd)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		profiles, err := selectedProfiles(cmd)
		if err != nil {
			return err
		}

		project, err := config.LoadProject()
		if err != nil {
			return err
		}
		return unlinkProfiles(project, profiles)
	}
	return cmd
}

func unlinkProfiles(project *config.Project, profiles []string) error {
	for _, source := range project.Sources {
		files, err := profilefiles.Discover(source.ResolvedPath, source.Config.Profiles, sourceSelectedProfiles(source, profiles))
		if err != nil {
			return fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		for _, file := range files {
			if err := unlinkProfileFile(project.HostRoot, source.ResolvedPath, file.RelPath); err != nil {
				return fmt.Errorf("unlink %s from source %q: %w", file.RelPath, source.Name, err)
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
