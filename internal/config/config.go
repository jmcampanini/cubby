package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	HostConfigFileName   = ".cubby.toml"
	SourceConfigFileName = "cubby.toml"
)

// DefaultHostConfig contains Cubby's default host repository config values.
var DefaultHostConfig = HostConfig{}

// DefaultSourceConfig contains Cubby's default source repository config values.
var DefaultSourceConfig = SourceConfig{}

// HostConfig is the host repository's .cubby.toml schema.
type HostConfig struct {
	Profiles        []string     `toml:"profiles" config:"profile" help:"profile to apply; repeatable or comma-separated"`
	IgnoreConflicts bool         `toml:"ignore_conflicts" config:"ignore-conflicts" help:"skip conflicting host paths instead of failing link"`
	CaseSensitive   bool         `toml:"case_sensitive" config:"case-sensitive" help:"treat projected host paths as case-sensitive"`
	Sources         []HostSource `toml:"source"`
}

// HostSource is one [[source]] entry in the host config.
type HostSource struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

// SourceConfig is a source repository's cubby.toml schema.
type SourceConfig struct {
	Profiles []string `toml:"profiles"`
	Ignore   []string `toml:"ignore"`
}

// NormalizeHostConfig returns cfg with profile defaults normalized.
func NormalizeHostConfig(cfg HostConfig) HostConfig {
	cfg.Profiles = NormalizeProfiles(cfg.Profiles)
	return cfg
}

// NormalizeSourceConfig returns cfg with declared profiles normalized.
func NormalizeSourceConfig(cfg SourceConfig) SourceConfig {
	cfg.Profiles = NormalizeProfiles(cfg.Profiles)
	return cfg
}

// ValidateSourceConfig normalizes and validates one source config.
func ValidateSourceConfig(sourceName string, cfg SourceConfig) (SourceConfig, error) {
	cfg = NormalizeSourceConfig(cfg)
	if len(cfg.Profiles) == 0 {
		return SourceConfig{}, fmt.Errorf("source %q declares no profiles", sourceName)
	}
	for _, raw := range cfg.Ignore {
		pattern := filepath.ToSlash(strings.TrimSpace(raw))
		if pattern == "" {
			continue
		}
		if !doublestar.ValidatePattern(pattern) {
			return SourceConfig{}, fmt.Errorf("source %q has invalid ignore pattern %q", sourceName, raw)
		}
	}
	return cfg, nil
}

// NormalizeProfiles trims whitespace, drops empty entries, and deduplicates while preserving first-seen order.
func NormalizeProfiles(values []string) []string {
	profiles := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		profile := strings.TrimSpace(value)
		if profile == "" {
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
