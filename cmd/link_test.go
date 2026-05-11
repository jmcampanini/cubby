package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinkProfileFileRefusesExistingRegularFile(t *testing.T) {
	root := t.TempDir()
	hostRoot := filepath.Join(root, "host")
	sourceRoot := filepath.Join(root, "src")
	rel := filepath.Join("nvim", "init.work.lua")
	mustWrite(t, filepath.Join(sourceRoot, rel), "-- source\n")
	mustWrite(t, filepath.Join(hostRoot, rel), "-- host\n")

	err := linkProfileFile(hostRoot, sourceRoot, rel)
	if err == nil {
		t.Fatalf("linkProfileFile() error = nil, want conflict")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("linkProfileFile() error = %q, want already exists", err)
	}
	if got := mustRead(t, filepath.Join(hostRoot, rel)); got != "-- host\n" {
		t.Fatalf("host file content = %q, want untouched", got)
	}
}

func TestLinkProfileFileRefusesUnexpectedSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	hostRoot := filepath.Join(root, "host")
	sourceRoot := filepath.Join(root, "src")
	otherRoot := filepath.Join(root, "other")
	rel := filepath.Join("nvim", "init.work.lua")
	hostPath := filepath.Join(hostRoot, rel)
	otherPath := filepath.Join(otherRoot, rel)
	mustWrite(t, filepath.Join(sourceRoot, rel), "-- source\n")
	mustWrite(t, otherPath, "-- other\n")
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(hostPath), err)
	}
	target, err := filepath.Rel(filepath.Dir(hostPath), otherPath)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if err := os.Symlink(target, hostPath); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", target, hostPath, err)
	}

	err = linkProfileFile(hostRoot, sourceRoot, rel)
	if err == nil {
		t.Fatalf("linkProfileFile() error = nil, want unexpected symlink conflict")
	}
	if !strings.Contains(err.Error(), "unexpected symlink") {
		t.Fatalf("linkProfileFile() error = %q, want unexpected symlink", err)
	}
	resolved, err := filepath.EvalSymlinks(hostPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", hostPath, err)
	}
	want, err := filepath.EvalSymlinks(otherPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", otherPath, err)
	}
	if resolved != want {
		t.Fatalf("host symlink resolved to %q, want still %q", resolved, want)
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

func mustRead(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
