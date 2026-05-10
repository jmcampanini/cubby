package profilefiles

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
)

// File is a profile-scoped file discovered in a source repository.
type File struct {
	RelPath string
	Profile string
}

// Discover walks root and returns regular files matching selected declared profiles.
func Discover(root string, declaredProfiles, selectedProfiles []string) ([]File, error) {
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

func selectedDeclaredProfiles(declaredProfiles, selectedProfiles []string) []string {
	declared := stringSet(declaredProfiles)
	var candidates map[string]struct{}
	if len(selectedProfiles) == 0 {
		candidates = declared
	} else {
		candidates = stringSet(selectedProfiles)
	}

	profiles := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for profile := range candidates {
		if _, ok := declared[profile]; !ok {
			continue
		}
		if _, ok := seen[profile]; ok {
			continue
		}
		seen[profile] = struct{}{}
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}
