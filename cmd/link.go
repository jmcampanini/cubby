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

func linkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Create symlinks for selected profiles",
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
		return linkProfiles(project, profiles)
	}
	return cmd
}

func linkProfiles(project *config.Project, profiles []string) error {
	for _, source := range project.Sources {
		files, err := profilefiles.Discover(source.ResolvedPath, source.Config.Profiles, sourceSelectedProfiles(source, profiles))
		if err != nil {
			return fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		for _, file := range files {
			if err := linkProfileFile(project.HostRoot, source.ResolvedPath, file.RelPath); err != nil {
				return fmt.Errorf("link %s from source %q: %w", file.RelPath, source.Name, err)
			}
		}
	}
	return nil
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
