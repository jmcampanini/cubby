package e2e_test

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestStatusAcceptanceReportsLinkedManagedSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "nvim", "init.work.lua"), filepath.Join(src, "nvim", "init.work.lua"))

	result := runCubby(t, bin, host, "status")
	if result.code != 0 {
		t.Fatalf("status code = %d, stderr = %s", result.code, result.stderr)
	}
	for _, want := range []string{"LINK nvim/init.work.lua", "source=src", "profile=work", "target=nvim/init.work.lua"} {
		assertContains(t, result.stdout, want)
	}
}

func TestStatusAcceptanceReportsDriftMarkers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "other.work"), "other\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "bad.work"), filepath.Join(src, "other.work"))

	result := runCubby(t, bin, host, "status")
	if result.code != 0 {
		t.Fatalf("status code = %d, stderr = %s", result.code, result.stderr)
	}
	for _, want := range []string{"DRIFT bad.work", "source=src", "target=other.work", "reasons=path mismatch"} {
		assertContains(t, result.stdout, want)
	}
}
