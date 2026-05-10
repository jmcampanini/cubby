package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	configloader "github.com/jmcampanini/go-config-loader"
)

// RegisteredSource is a host source entry after path resolution and source config loading.
type RegisteredSource struct {
	HostSource
	ResolvedPath string
	Config       SourceConfig
}

// Project is the loaded Cubby configuration rooted at a host repository.
type Project struct {
	HostRoot string
	Host     HostConfig
	Sources  []RegisteredSource
}

// LoadProject discovers the host root from startDir and loads the host config plus all registered sources.
func LoadProject(startDir string) (*Project, error) {
	hostRoot, err := DiscoverHostRoot(startDir)
	if err != nil {
		return nil, err
	}

	hostFile := filepath.Join(hostRoot, HostConfigFileName)
	hostCfg, err := loadRequiredFile(hostFile, DefaultHostConfig)
	if err != nil {
		return nil, fmt.Errorf("load host config %q: %w", hostFile, err)
	}
	if len(hostCfg.Sources) == 0 {
		return nil, fmt.Errorf("host config %q has no [[source]] entries", hostFile)
	}

	project := &Project{
		HostRoot: hostRoot,
		Host:     hostCfg,
		Sources:  make([]RegisteredSource, 0, len(hostCfg.Sources)),
	}

	for i, source := range hostCfg.Sources {
		registered, err := loadRegisteredSource(hostRoot, i, source)
		if err != nil {
			return nil, err
		}
		project.Sources = append(project.Sources, registered)
	}
	if len(project.DeclaredProfiles()) == 0 {
		return nil, fmt.Errorf("registered sources declare no profiles")
	}

	return project, nil
}

// DiscoverHostRoot walks upward from startDir until it finds .cubby.toml.
func DiscoverHostRoot(startDir string) (string, error) {
	if startDir == "" {
		startDir = "."
	}
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start directory %q: %w", startDir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat start directory %q: %w", abs, err)
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		candidate := filepath.Join(abs, HostConfigFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return abs, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat host config %q: %w", candidate, err)
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("missing host config %s: no %s found from current directory upward", HostConfigFileName, HostConfigFileName)
		}
		abs = parent
	}
}

// DeclaredProfiles returns the sorted union of profiles declared by registered sources.
func (p *Project) DeclaredProfiles() []string {
	seen := make(map[string]struct{})
	for _, source := range p.Sources {
		for _, profile := range source.Config.Profiles {
			profile = strings.TrimSpace(profile)
			if profile == "" {
				continue
			}
			seen[profile] = struct{}{}
		}
	}

	profiles := make([]string, 0, len(seen))
	for profile := range seen {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles
}

func loadRegisteredSource(hostRoot string, index int, source HostSource) (RegisteredSource, error) {
	name := source.Name
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("#%d", index+1)
	}
	if strings.TrimSpace(source.Path) == "" {
		return RegisteredSource{}, fmt.Errorf("source %q is missing path", name)
	}

	resolvedPath, err := ResolveSourcePath(hostRoot, source.Path)
	if err != nil {
		return RegisteredSource{}, fmt.Errorf("resolve path for source %q: %w", name, err)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return RegisteredSource{}, fmt.Errorf("source %q path does not exist: %s", name, resolvedPath)
		}
		return RegisteredSource{}, fmt.Errorf("stat source %q path %q: %w", name, resolvedPath, err)
	}
	if !info.IsDir() {
		return RegisteredSource{}, fmt.Errorf("source %q path is not a directory: %s", name, resolvedPath)
	}

	sourceFile := filepath.Join(resolvedPath, SourceConfigFileName)
	sourceCfg, err := loadRequiredFile(sourceFile, DefaultSourceConfig)
	if err != nil {
		return RegisteredSource{}, fmt.Errorf("load source config for source %q at %q: %w", name, sourceFile, err)
	}
	if len(nonEmptyStrings(sourceCfg.Profiles)) == 0 {
		return RegisteredSource{}, fmt.Errorf("source %q declares no profiles", name)
	}

	return RegisteredSource{
		HostSource:   source,
		ResolvedPath: resolvedPath,
		Config:       sourceCfg,
	}, nil
}

// ResolveSourcePath resolves absolute paths, ~/ paths, and host-root-relative paths.
func ResolveSourcePath(hostRoot, sourcePath string) (string, error) {
	expanded, err := expandHome(sourcePath)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(hostRoot, expanded)
	}
	return filepath.Clean(expanded), nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func loadRequiredFile[T any](path string, defaults T) (T, error) {
	loader, err := configloader.NewRequiredFileLoader[T](path)
	if err != nil {
		return defaults, err
	}
	cfg, _, err := configloader.Load(defaults, loader)
	if err != nil {
		return defaults, err
	}
	return cfg, nil
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
