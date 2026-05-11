package linkops

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRelativeTargetForNestedHostPath(t *testing.T) {
	root := t.TempDir()
	hostPath := filepath.Join(root, "host", "nvim", "init.work.lua")
	sourcePath := filepath.Join(root, "src", "nvim", "init.work.lua")

	got, err := RelativeTarget(hostPath, sourcePath)
	if err != nil {
		t.Fatalf("RelativeTarget() error = %v", err)
	}
	want := filepath.Join("..", "..", "src", "nvim", "init.work.lua")
	if got != want {
		t.Fatalf("RelativeTarget() = %q, want %q", got, want)
	}
}

func TestPointsToDetectsCorrectRelativeSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "src", "nvim", "init.work.lua")
	link := filepath.Join(root, "host", "nvim", "init.work.lua")
	mustWrite(t, target, "-- work\n")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(link), err)
	}
	rel, err := RelativeTarget(link, target)
	if err != nil {
		t.Fatalf("RelativeTarget() error = %v", err)
	}
	if err := os.Symlink(rel, link); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", rel, link, err)
	}

	ok, err := PointsTo(link, target)
	if err != nil {
		t.Fatalf("PointsTo() error = %v", err)
	}
	if !ok {
		t.Fatalf("PointsTo() = false, want true")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
