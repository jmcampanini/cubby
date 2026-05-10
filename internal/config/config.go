package config

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
	Sources []HostSource `toml:"source"`
}

// HostSource is one [[source]] entry in the host config.
type HostSource struct {
	Name            string   `toml:"name"`
	Path            string   `toml:"path"`
	Profiles        []string `toml:"profiles"`
	IgnoreConflicts bool     `toml:"ignore_conflicts"`
}

// SourceConfig is a source repository's cubby.toml schema.
type SourceConfig struct {
	Profiles []string `toml:"profiles"`
	Ignore   []string `toml:"ignore"`
}
