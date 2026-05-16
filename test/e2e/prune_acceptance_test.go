package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPruneAcceptanceRemovesOnlyDanglingManagedSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	other := filepath.Join(tmp, "other")
	managed := filepath.Join(host, "nvim", "init.work.lua")
	valid := filepath.Join(host, "valid.work")
	unmanaged := filepath.Join(host, "outside.work")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "valid.work"), "valid\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "stale\n")
	if err := os.Remove(filepath.Join(src, "nvim", "init.work.lua")); err != nil {
		t.Fatalf("Remove stale source error = %v", err)
	}
	mustWrite(t, filepath.Join(other, "placeholder"), "other\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, managed, filepath.Join(src, "nvim", "init.work.lua"))
	mustSymlink(t, valid, filepath.Join(src, "valid.work"))
	mustSymlink(t, unmanaged, filepath.Join(other, "gone.work"))

	result := runCubby(t, bin, host, "prune")
	if result.code != 0 {
		t.Fatalf("prune code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "nvim/init.work.lua")
	assertNotExist(t, managed)
	assertSymlinkExists(t, valid)
	assertSymlinkExists(t, unmanaged)
}
