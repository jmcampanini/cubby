package gitignore

import (
	"fmt"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
)

// MissingProjectPatterns reports required patterns absent from the host gitignore.
func MissingProjectPatterns(project *config.Project) ([]string, error) {
	profilePatterns := RequiredPatterns(project.DeclaredProfiles())
	required := append([]string{"/" + config.HostConfigFileName}, profilePatterns...)
	gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
	missing, err := MissingPatternsFile(gitignorePath, required)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", gitignorePath, err)
	}
	return missing, nil
}
