package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/profilefiles"
)

type discoveredProfileFiles struct {
	source config.RegisteredSource
	files  []profilefiles.File
}

func discoverProfileFiles(project *config.Project, profiles []string) ([]discoveredProfileFiles, error) {
	discovered := make([]discoveredProfileFiles, 0, len(project.Sources))
	for _, source := range project.Sources {
		discoveryRoot, err := sourceDiscoveryRoot(source.ResolvedPath)
		if err != nil {
			return nil, fmt.Errorf("resolve discovery root for source %q: %w", source.Name, err)
		}
		files, err := profilefiles.Discover(discoveryRoot, source.Config.Profiles, sourceSelectedProfiles(source, profiles), source.Config.Ignore)
		if err != nil {
			return nil, fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		discovered = append(discovered, discoveredProfileFiles{source: source, files: files})
	}
	return discovered, nil
}

func sourceDiscoveryRoot(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return root, nil
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return resolved, nil
}
