package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	configloader "github.com/jmcampanini/go-config-loader"
)

var validSourceName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

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

// LoadProject loads the host config from the current directory plus all registered sources.
func LoadProject() (*Project, error) {
	hostRoot, err := CurrentHostRoot()
	if err != nil {
		return nil, err
	}

	hostFile := filepath.Join(hostRoot, HostConfigFileName)
	hostCfg, err := LoadHostConfigFile(hostFile)
	if err != nil {
		return nil, fmt.Errorf("load host config %q: %w", hostFile, err)
	}

	return LoadProjectWithHostConfig(hostRoot, hostCfg)
}

// CurrentHostRoot returns the current directory as an absolute host root after verifying it contains the host config.
func CurrentHostRoot() (string, error) {
	hostRoot, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	hostRoot, err = filepath.Abs(hostRoot)
	if err != nil {
		return "", fmt.Errorf("resolve current directory %q: %w", hostRoot, err)
	}
	hostRoot = filepath.Clean(hostRoot)

	hostFile := filepath.Join(hostRoot, HostConfigFileName)
	info, err := os.Stat(hostFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found in current directory; run cubby from the host repo root", HostConfigFileName)
		}
		return "", fmt.Errorf("stat host config %q: %w", hostFile, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("host config %q is a directory", hostFile)
	}
	return hostRoot, nil
}

// LoadHostConfigFile loads and normalizes one required host config file.
func LoadHostConfigFile(path string) (HostConfig, error) {
	hostCfg, err := loadRequiredFile(path, DefaultHostConfig)
	if err != nil {
		return HostConfig{}, err
	}
	return NormalizeHostConfig(hostCfg), nil
}

// LoadProjectWithHostConfig loads registered sources using an already-effective host config.
func LoadProjectWithHostConfig(hostRoot string, hostCfg HostConfig) (*Project, error) {
	hostRoot, err := filepath.Abs(hostRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve host root %q: %w", hostRoot, err)
	}
	hostRoot = filepath.Clean(hostRoot)
	hostCfg = NormalizeHostConfig(hostCfg)
	hostFile := filepath.Join(hostRoot, HostConfigFileName)
	if len(hostCfg.Sources) == 0 {
		return nil, fmt.Errorf("host config %q has no [[source]] entries", hostFile)
	}
	if err := validateHostSources(hostCfg.Sources); err != nil {
		return nil, err
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

// DeclaredProfiles returns the sorted union of profiles declared by registered sources.
func (p *Project) DeclaredProfiles() []string {
	seen := make(map[string]struct{})
	for _, source := range p.Sources {
		for _, profile := range NormalizeProfiles(source.Config.Profiles) {
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

func validateHostSources(sources []HostSource) error {
	seen := make(map[string]struct{}, len(sources))
	for i, source := range sources {
		name := source.Name
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("source entry %d is missing name (source name required)", i+1)
		}
		if !validSourceName.MatchString(name) {
			return fmt.Errorf("source name %q is invalid; must match %s", name, validSourceName.String())
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate source name %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func loadRegisteredSource(hostRoot string, _ int, source HostSource) (RegisteredSource, error) {
	name := source.Name
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
	sourceCfg = NormalizeSourceConfig(sourceCfg)
	if len(sourceCfg.Profiles) == 0 {
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
