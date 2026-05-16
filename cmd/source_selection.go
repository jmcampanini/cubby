package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
)

func selectSource(project *config.Project, requested string, requestedSet bool) (config.RegisteredSource, error) {
	if project == nil {
		return config.RegisteredSource{}, fmt.Errorf("project is nil")
	}

	requested = strings.TrimSpace(requested)
	if requestedSet {
		if requested == "" {
			return config.RegisteredSource{}, fmt.Errorf("--source must not be empty")
		}
		for _, source := range project.Sources {
			if source.Name == requested {
				return source, nil
			}
		}
		return config.RegisteredSource{}, fmt.Errorf("unknown source %q; known sources: %s", requested, knownSourceNames(project))
	}

	switch len(project.Sources) {
	case 0:
		return config.RegisteredSource{}, fmt.Errorf("no sources registered")
	case 1:
		return project.Sources[0], nil
	default:
		return config.RegisteredSource{}, fmt.Errorf("multiple sources registered; specify --source (known sources: %s)", knownSourceNames(project))
	}
}

func knownSourceNames(project *config.Project) string {
	names := make([]string, 0, len(project.Sources))
	for _, source := range project.Sources {
		names = append(names, source.Name)
	}
	return quotedList(names)
}
