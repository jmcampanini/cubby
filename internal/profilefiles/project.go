package profilefiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/linkops"
)

// SourceFiles holds the profile files found in one registered source.
type SourceFiles struct {
	Source config.RegisteredSource
	Files  []File
}

// DiscoverProject finds selected profile files in every registered source, in registration order.
func DiscoverProject(project *config.Project, profiles []string) ([]SourceFiles, error) {
	discovered := make([]SourceFiles, 0, len(project.Sources))
	for _, source := range project.Sources {
		discoveryRoot, err := sourceDiscoveryRoot(source.ResolvedPath)
		if err != nil {
			return nil, fmt.Errorf("resolve discovery root for source %q: %w", source.Name, err)
		}
		files, err := Discover(discoveryRoot, source.Config.Profiles, sourceSelectedProfiles(source, profiles), source.Config.Ignore)
		if err != nil {
			return nil, fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		discovered = append(discovered, SourceFiles{Source: source, Files: files})
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

func sourceSelectedProfiles(source config.RegisteredSource, selected []string) []string {
	allowed := make(map[string]struct{})
	for _, profile := range config.NormalizeProfiles(source.Config.Profiles) {
		allowed[profile] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}

	profiles := make([]string, 0, len(selected))
	for _, profile := range config.NormalizeProfiles(selected) {
		if _, ok := allowed[profile]; ok {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// LinkSources converts discovered files into link planning inputs.
func LinkSources(discovered []SourceFiles) []linkops.SourceFiles {
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
