package profilefiles

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/jmcampanini/cubby/internal/config"
)

// File is a profile-scoped file discovered in a source repository.
type File struct {
	RelPath string
	Profile string
}

// Discover walks root and returns regular files matching selected declared profiles.
func Discover(root string, declaredProfiles, selectedProfiles, ignore []string) ([]File, error) {
	ignoreRules, err := compileIgnoreRules(ignore)
	if err != nil {
		return nil, err
	}

	profiles := selectedDeclaredProfiles(declaredProfiles, selectedProfiles)
	if len(profiles) == 0 {
		return nil, nil
	}

	var files []File
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == config.SourceConfigFileName {
			return nil
		}

		relSlash := filepath.ToSlash(rel)
		ignored, err := ignoredByAny(ignoreRules, relSlash, d.Name())
		if err != nil {
			return err
		}
		if ignored {
			return nil
		}

		base := d.Name()
		for _, profile := range profiles {
			if MatchBasename(base, profile) {
				files = append(files, File{RelPath: rel, Profile: profile})
				break
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].RelPath == files[j].RelPath {
			return files[i].Profile < files[j].Profile
		}
		return files[i].RelPath < files[j].RelPath
	})
	return files, nil
}

// Ignored reports whether relPath is excluded by ignore rules.
func Ignored(relPath string, ignore []string) (bool, error) {
	ignoreRules, err := compileIgnoreRules(ignore)
	if err != nil {
		return false, err
	}
	relSlash := filepath.ToSlash(filepath.Clean(relPath))
	return ignoredByAny(ignoreRules, relSlash, filepath.Base(relPath))
}

// MatchBasename reports whether base contains profile as an exact dot segment.
func MatchBasename(base, profile string) bool {
	profile = strings.TrimSpace(profile)
	if base == "" || profile == "" {
		return false
	}

	marker := "." + profile
	for start := strings.Index(base, marker); start >= 0; {
		after := start + len(marker)
		if after == len(base) {
			if start > 0 {
				return true
			}
		} else if base[after] == '.' {
			return true
		}

		nextFrom := start + 1
		if nextFrom >= len(base) {
			break
		}
		next := strings.Index(base[nextFrom:], marker)
		if next < 0 {
			break
		}
		start = nextFrom + next
	}
	return false
}

type ignoreRule struct {
	pattern      string
	basenameOnly bool
}

func compileIgnoreRules(ignore []string) ([]ignoreRule, error) {
	rules := make([]ignoreRule, 0, len(ignore))
	for _, raw := range ignore {
		pattern := filepath.ToSlash(strings.TrimSpace(raw))
		if pattern == "" {
			continue
		}
		if !doublestar.ValidatePattern(pattern) {
			return nil, fmt.Errorf("invalid ignore pattern %q", raw)
		}
		rules = append(rules, ignoreRule{
			pattern:      pattern,
			basenameOnly: !strings.Contains(pattern, "/"),
		})
	}
	return rules, nil
}

func ignoredByAny(rules []ignoreRule, relSlash, base string) (bool, error) {
	for _, rule := range rules {
		name := relSlash
		if rule.basenameOnly {
			name = base
		}
		matched, err := doublestar.Match(rule.pattern, name)
		if err != nil {
			return false, fmt.Errorf("invalid ignore pattern %q: %w", rule.pattern, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func selectedDeclaredProfiles(declaredProfiles, selectedProfiles []string) []string {
	declared := stringSet(declaredProfiles)
	if len(declared) == 0 {
		return nil
	}

	profiles := make([]string, 0, len(selectedProfiles))
	seen := make(map[string]struct{}, len(selectedProfiles))
	for _, profile := range config.NormalizeProfiles(selectedProfiles) {
		if _, ok := declared[profile]; !ok {
			continue
		}
		if _, ok := seen[profile]; ok {
			continue
		}
		seen[profile] = struct{}{}
		profiles = append(profiles, profile)
	}
	return profiles
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range config.NormalizeProfiles(values) {
		set[value] = struct{}{}
	}
	return set
}
