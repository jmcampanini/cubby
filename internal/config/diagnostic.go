package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiagnosticSourceRoot is a host source entry whose path could be resolved lexically.
type DiagnosticSourceRoot struct {
	Name         string
	ResolvedPath string
	Order        int
}

// SourceIssue describes a registered source that could not be used for diagnostics.
type SourceIssue struct {
	Name string
	Path string
	Kind string
	Err  error
}

func (i SourceIssue) Error() string {
	if i.Err == nil {
		return i.Kind
	}
	return i.Err.Error()
}

// DiagnosticProject is a best-effort project load for health checks.
type DiagnosticProject struct {
	HostRoot     string
	Host         HostConfig
	Sources      []RegisteredSource
	SourceRoots  []DiagnosticSourceRoot
	SourceIssues []SourceIssue
}

// LoadProjectDiagnostics loads host config strictly, then loads sources best-effort.
func LoadProjectDiagnostics() (*DiagnosticProject, error) {
	hostRoot, err := CurrentHostRoot()
	if err != nil {
		return nil, err
	}

	hostFile := filepath.Join(hostRoot, HostConfigFileName)
	hostCfg, err := LoadHostConfigFile(hostFile)
	if err != nil {
		return nil, fmt.Errorf("load host config %q: %w", hostFile, err)
	}

	return LoadProjectDiagnosticsWithHostConfig(hostRoot, hostCfg)
}

// LoadProjectDiagnosticsWithHostConfig loads valid sources and records invalid ones.
func LoadProjectDiagnosticsWithHostConfig(hostRoot string, hostCfg HostConfig) (*DiagnosticProject, error) {
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

	project := &DiagnosticProject{HostRoot: hostRoot, Host: hostCfg}
	for i, source := range hostCfg.Sources {
		registered, root, issue := loadRegisteredSourceDiagnostic(hostRoot, i, source)
		if root != nil {
			project.SourceRoots = append(project.SourceRoots, *root)
		}
		if issue != nil {
			project.SourceIssues = append(project.SourceIssues, *issue)
			continue
		}
		project.Sources = append(project.Sources, registered)
	}
	return project, nil
}

// Project returns the valid-source subset as a strict Project shape for shared helpers.
func (p *DiagnosticProject) Project() *Project {
	return &Project{HostRoot: p.HostRoot, Host: p.Host, Sources: p.Sources}
}

func loadRegisteredSourceDiagnostic(hostRoot string, order int, source HostSource) (RegisteredSource, *DiagnosticSourceRoot, *SourceIssue) {
	name := source.Name
	issue := func(kind string, path string, err error) (RegisteredSource, *DiagnosticSourceRoot, *SourceIssue) {
		return RegisteredSource{}, nil, &SourceIssue{Name: name, Path: path, Kind: kind, Err: err}
	}
	issueWithRoot := func(kind string, path string, err error) (RegisteredSource, *DiagnosticSourceRoot, *SourceIssue) {
		return RegisteredSource{}, &DiagnosticSourceRoot{Name: name, ResolvedPath: path, Order: order}, &SourceIssue{Name: name, Path: path, Kind: kind, Err: err}
	}
	if source.Path == "" {
		return issue("missing_path", "", fmt.Errorf("source %q is missing path", name))
	}

	resolvedPath, err := ResolveSourcePath(hostRoot, source.Path)
	if err != nil {
		return issue("invalid_path", source.Path, fmt.Errorf("resolve path for source %q: %w", name, err))
	}
	root := &DiagnosticSourceRoot{Name: name, ResolvedPath: resolvedPath, Order: order}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return issueWithRoot("missing_path", resolvedPath, fmt.Errorf("source %q path does not exist: %s", name, resolvedPath))
		}
		return issueWithRoot("invalid_path", resolvedPath, fmt.Errorf("stat source %q path %q: %w", name, resolvedPath, err))
	}
	if !info.IsDir() {
		return issueWithRoot("not_directory", resolvedPath, fmt.Errorf("source %q path is not a directory: %s", name, resolvedPath))
	}

	sourceFile := filepath.Join(resolvedPath, SourceConfigFileName)
	sourceCfg, err := LoadSourceConfigFile(sourceFile, name)
	if err != nil {
		return RegisteredSource{}, root, &SourceIssue{Name: name, Path: resolvedPath, Kind: "invalid_config", Err: fmt.Errorf("load source config for source %q at %q: %w", name, sourceFile, err)}
	}

	return RegisteredSource{HostSource: source, ResolvedPath: resolvedPath, Config: sourceCfg}, root, nil
}
