package config

import "strings"

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
	Profiles []string     `toml:"profiles" config:"profile" help:"profile to apply; repeatable or comma-separated"`
	Sources  []HostSource `toml:"source"`
}

// HostSource is one [[source]] entry in the host config.
type HostSource struct {
	Name            string `toml:"name"`
	Path            string `toml:"path"`
	IgnoreConflicts bool   `toml:"ignore_conflicts"`
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
