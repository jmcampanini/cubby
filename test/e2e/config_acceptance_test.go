package e2e_test

import (
	"path/filepath"
	"testing"
)

func TestConfigErrorCases(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()

	t.Run("missing host config", func(t *testing.T) {
		dir := filepath.Join(tmp, "missing-host")
		mustMkdir(t, dir)
		result := runCubby(t, bin, dir, "gitignore", "check")
		assertFailureContains(t, result, ".cubby.toml")
	})

	t.Run("missing source path", func(t *testing.T) {
		host := filepath.Join(tmp, "missing-source-path")
		mustMkdir(t, host)
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \"./does-not-exist\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "path does not exist")
	})

	t.Run("missing source config", func(t *testing.T) {
		host := filepath.Join(tmp, "missing-source-config-host")
		src := filepath.Join(tmp, "missing-source-config-src")
		mustMkdir(t, host)
		mustMkdir(t, src)
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "cubby.toml")
	})

	t.Run("source config with no declared profiles", func(t *testing.T) {
		host := filepath.Join(tmp, "no-profiles-host")
		src := filepath.Join(tmp, "no-profiles-src")
		mustMkdir(t, host)
		mustMkdir(t, src)
		mustWrite(t, filepath.Join(src, "cubby.toml"), "ignore = []\n")
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "declares no profiles")
	})
}
