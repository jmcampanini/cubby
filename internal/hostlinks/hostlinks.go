package hostlinks

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/profilefiles"
)

const (
	ReasonDangling     = "dangling"
	ReasonPathMismatch = "path mismatch"
	ReasonUnknown      = "unknown profile"
	ReasonIgnored      = "ignored"
)

// ManagedLink is a symlink in the host repo whose target belongs to a registered source.
type ManagedLink struct {
	HostPath      string
	HostRelPath   string
	RawTarget     string
	TargetPath    string
	SourceName    string
	SourceRoot    string
	SourceRelPath string
	Profile       string
	TargetExists  bool
	DriftReasons  []string
}

type sourceRoot struct {
	name      string
	config    config.SourceConfig
	hasConfig bool
	root      string
	matchRoot string
	order     int
}

// Discover walks hostRoot and returns symlinks managed by registered sources.
func Discover(hostRoot string, sources []config.RegisteredSource) ([]ManagedLink, error) {
	return discover(hostRoot, sourceRootsFromRegistered(sources, nil))
}

// DiscoverDiagnostics walks hostRoot using valid sources plus diagnostic roots for dangling missing-source links.
func DiscoverDiagnostics(hostRoot string, sources []config.RegisteredSource, diagnosticRoots []config.DiagnosticSourceRoot) ([]ManagedLink, error) {
	return discover(hostRoot, sourceRootsFromRegistered(sources, diagnosticRoots))
}

func discover(hostRoot string, roots []sourceRoot) ([]ManagedLink, error) {
	hostRoot, err := filepath.Abs(hostRoot)
	if err != nil {
		return nil, err
	}
	hostRoot = filepath.Clean(hostRoot)

	var links []ManagedLink
	if err := filepath.WalkDir(hostRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		path = filepath.Clean(path)
		if d.IsDir() {
			if d.Name() == ".git" && path != hostRoot {
				return filepath.SkipDir
			}
			if path != hostRoot && isRegisteredSourceDir(path, roots) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}

		link, managed, err := classify(hostRoot, path, roots)
		if err != nil {
			return err
		}
		if managed {
			links = append(links, link)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(links, func(i, j int) bool {
		if links[i].HostRelPath == links[j].HostRelPath {
			return links[i].SourceName < links[j].SourceName
		}
		return links[i].HostRelPath < links[j].HostRelPath
	})
	return links, nil
}

func sourceRootsFromRegistered(sources []config.RegisteredSource, diagnosticRoots []config.DiagnosticSourceRoot) []sourceRoot {
	orderByKey := make(map[string]int, len(diagnosticRoots))
	for _, root := range diagnosticRoots {
		clean := filepath.Clean(root.ResolvedPath)
		key := rootKey(root.Name, clean)
		orderByKey[key] = root.Order
	}

	roots := make([]sourceRoot, 0, len(sources)+len(diagnosticRoots))
	valid := make(map[string]struct{}, len(sources))
	for i, source := range sources {
		root, err := filepath.Abs(source.ResolvedPath)
		if err != nil {
			root = source.ResolvedPath
		}
		root = filepath.Clean(root)
		key := rootKey(source.Name, root)
		order, ok := orderByKey[key]
		if !ok {
			order = i
		}
		roots = append(roots, newSourceRoot(source.Name, root, source.Config, true, order))
		valid[key] = struct{}{}
	}
	seenInvalid := make(map[string]struct{}, len(diagnosticRoots))
	for _, root := range diagnosticRoots {
		clean := filepath.Clean(root.ResolvedPath)
		key := rootKey(root.Name, clean)
		if _, ok := valid[key]; ok {
			continue
		}
		if _, ok := seenInvalid[key]; ok {
			continue
		}
		seenInvalid[key] = struct{}{}
		roots = append(roots, newSourceRoot(root.Name, clean, config.SourceConfig{}, false, root.Order))
	}
	return roots
}

func newSourceRoot(name, root string, cfg config.SourceConfig, hasConfig bool, order int) sourceRoot {
	root = filepath.Clean(root)
	matchRoot := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		matchRoot = filepath.Clean(resolved)
	}
	return sourceRoot{name: name, config: cfg, hasConfig: hasConfig, root: root, matchRoot: matchRoot, order: order}
}

func rootKey(name, root string) string {
	return name + "\x00" + filepath.Clean(root)
}

func classify(hostRoot, hostPath string, roots []sourceRoot) (ManagedLink, bool, error) {
	rawTarget, err := os.Readlink(hostPath)
	if err != nil {
		return ManagedLink{}, false, err
	}
	targetPath := rawTarget
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(filepath.Dir(hostPath), targetPath)
	}
	targetPath = filepath.Clean(targetPath)

	targetExists := true
	matchTarget := targetPath
	if resolved, err := filepath.EvalSymlinks(targetPath); err == nil {
		matchTarget = filepath.Clean(resolved)
	} else if os.IsNotExist(err) {
		targetExists = false
	} else {
		// Broken intermediate components and permission errors are still treated lexically.
		if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
			targetExists = false
		}
	}

	owner, ok := owningSource(matchTarget, roots, targetExists)
	if !ok {
		return ManagedLink{}, false, nil
	}
	sourceRel, err := filepath.Rel(owner.matchRoot, matchTarget)
	if err != nil {
		return ManagedLink{}, false, err
	}
	hostRel, err := filepath.Rel(hostRoot, hostPath)
	if err != nil {
		return ManagedLink{}, false, err
	}

	link := ManagedLink{
		HostPath:      hostPath,
		HostRelPath:   filepath.Clean(hostRel),
		RawTarget:     rawTarget,
		TargetPath:    matchTarget,
		SourceName:    owner.name,
		SourceRoot:    owner.root,
		SourceRelPath: filepath.Clean(sourceRel),
		TargetExists:  targetExists,
		DriftReasons:  nil,
	}
	if owner.hasConfig {
		link.Profile = inferProfile(filepath.Base(link.SourceRelPath), owner.config.Profiles)
	}

	if !targetExists {
		link.DriftReasons = append(link.DriftReasons, ReasonDangling)
	}
	if filepath.Clean(link.HostRelPath) != filepath.Clean(link.SourceRelPath) {
		link.DriftReasons = append(link.DriftReasons, ReasonPathMismatch)
	}
	if owner.hasConfig {
		if link.Profile == "" {
			link.DriftReasons = append(link.DriftReasons, ReasonUnknown)
		}
		ignored, err := profilefiles.Ignored(link.SourceRelPath, owner.config.Ignore)
		if err != nil {
			return ManagedLink{}, false, err
		}
		if ignored {
			link.DriftReasons = append(link.DriftReasons, ReasonIgnored)
		}
	}

	return link, true, nil
}

func owningSource(target string, roots []sourceRoot, targetExists bool) (sourceRoot, bool) {
	var best sourceRoot
	bestLen := -1
	ok := false
	for _, root := range roots {
		if targetExists && !root.hasConfig {
			continue
		}
		matchRoot := root.matchRoot
		if !targetExists {
			matchRoot = root.root
		}
		if !insideRoot(matchRoot, target) {
			continue
		}
		rootLen := len(splitPath(matchRoot))
		if !ok || rootLen > bestLen || (rootLen == bestLen && root.order < best.order) {
			best = root
			if !targetExists {
				best.matchRoot = root.root
			}
			bestLen = rootLen
			ok = true
		}
	}
	return best, ok
}

func insideRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !startsWithDotDot(rel))
}

func startsWithDotDot(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == ".."+string(filepath.Separator)
}

func splitPath(path string) []string {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, filepath.VolumeName(path))
	path = strings.Trim(path, string(filepath.Separator))
	if path == "" {
		return nil
	}
	return strings.Split(path, string(filepath.Separator))
}

func inferProfile(base string, profiles []string) string {
	for _, profile := range config.NormalizeProfiles(profiles) {
		if profilefiles.MatchBasename(base, profile) {
			return profile
		}
	}
	return ""
}

func isRegisteredSourceDir(path string, roots []sourceRoot) bool {
	for _, root := range roots {
		if filepath.Clean(path) == root.root || filepath.Clean(path) == root.matchRoot {
			return true
		}
	}
	return false
}
