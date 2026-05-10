package profilefiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchBasenameForValidProfileForms(t *testing.T) {
	tests := []struct {
		base string
		want bool
	}{
		{base: "init.work.lua", want: true},
		{base: "archive.work.tar.gz", want: true},
		{base: "Makefile.work", want: true},
		{base: ".gitignore.work", want: true},
		{base: ".zshrc.work", want: true},
		{base: ".zshrc.work.local", want: true},
		{base: ".work.local", want: true},
		{base: ".work", want: false},
		{base: "script.workbench.sh", want: false},
		{base: "thing.homework", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			if got := MatchBasename(tt.base, "work"); got != tt.want {
				t.Fatalf("MatchBasename(%q, work) = %v, want %v", tt.base, got, tt.want)
			}
		})
	}
}

func TestDiscoverPreservesNestedPathsAndIgnoresUndeclaredLookalikes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(root, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(root, "nvim", "init.test.lua"), "-- test\n")
	mustWrite(t, filepath.Join(root, "script.workbench.sh"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(root, ".work"), "unsupported\n")
	mustWrite(t, filepath.Join(root, ".git", "ignored.work"), "ignored\n")

	files, err := Discover(root, []string{"work"}, []string{"work", "test"})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Discover() = %#v, want one file", files)
	}
	want := filepath.Join("nvim", "init.work.lua")
	if files[0].RelPath != want {
		t.Fatalf("RelPath = %q, want %q", files[0].RelPath, want)
	}
	if files[0].Profile != "work" {
		t.Fatalf("Profile = %q, want work", files[0].Profile)
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
