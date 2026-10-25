// Package projectfiles discovers selected files across registered sources.
package projectfiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/linkops"
	"github.com/jmcampanini/cubby/internal/profilefiles"
)

// Discovery holds the profile files found in one registered source.
type Discovery struct {
	Source config.RegisteredSource
	Files  []profilefiles.File
}

// Discover finds selected profile files in registration order.
func Discover(project *config.Project, profiles []string) ([]Discovery, error) {
	discovered := make([]Discovery, 0, len(project.Sources))
	for _, source := range project.Sources {
		discoveryRoot, err := sourceDiscoveryRoot(source.ResolvedPath)
		if err != nil {
			return nil, fmt.Errorf("resolve discovery root for source %q: %w", source.Name, err)
		}
		files, err := profilefiles.Discover(discoveryRoot, source.Config.Profiles, config.SourceSelectedProfiles(source, profiles), source.Config.Ignore)
		if err != nil {
			return nil, fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		discovered = append(discovered, Discovery{Source: source, Files: files})
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

// LinkSources converts discovered files into link planning inputs.
func LinkSources(discovered []Discovery) []linkops.SourceFiles {
	sources := make([]linkops.SourceFiles, 0, len(discovered))
	for _, item := range discovered {
		relPaths := make([]string, 0, len(item.Files))
		for _, file := range item.Files {
			relPaths = append(relPaths, file.RelPath)
		}
		sources = append(sources, linkops.SourceFiles{
			Name:     item.Source.Name,
			Root:     item.Source.ResolvedPath,
			RelPaths: relPaths,
		})
	}
	return sources
}
