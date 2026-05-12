package cmd

import (
	"fmt"

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
		files, err := profilefiles.Discover(source.ResolvedPath, source.Config.Profiles, sourceSelectedProfiles(source, profiles), source.Config.Ignore)
		if err != nil {
			return nil, fmt.Errorf("discover profile files for source %q: %w", source.Name, err)
		}
		discovered = append(discovered, discoveredProfileFiles{source: source, files: files})
	}
	return discovered, nil
}
