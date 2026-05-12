package profilefiles

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDiscoverWithEmptySelectionReturnsNoFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "nvim", "init.work.lua"), "-- work\n")

	files, err := Discover(root, []string{"work"}, nil, nil)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("Discover() = %#v, want no files", files)
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

	files, err := Discover(root, []string{"work"}, []string{"work", "test"}, nil)
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

func TestDiscoverAppliesIgnoreRules(t *testing.T) {
	tests := []struct {
		name        string
		ignore      []string
		files       []string
		wantRelPath []string
	}{
		{
			name:        "exact source relative path",
			ignore:      []string{"nvim/init.work.lua"},
			files:       []string{"nvim/init.work.lua", "zsh/zshrc.work"},
			wantRelPath: []string{filepath.Join("zsh", "zshrc.work")},
		},
		{
			name:        "basename exact anywhere",
			ignore:      []string{"init.work.lua"},
			files:       []string{"nvim/init.work.lua", "other/init.work.lua", "nvim/keep.work.lua"},
			wantRelPath: []string{filepath.Join("nvim", "keep.work.lua")},
		},
		{
			name:        "basename glob anywhere",
			ignore:      []string{"*.draft.*"},
			files:       []string{"nvim/init.work.draft.lua", "nvim/init.work.lua"},
			wantRelPath: []string{filepath.Join("nvim", "init.work.lua")},
		},
		{
			name:        "recursive doublestar glob",
			ignore:      []string{"**/*.draft.*"},
			files:       []string{"root.work.draft.lua", "nvim/init.work.draft.lua", "nvim/init.work.lua"},
			wantRelPath: []string{filepath.Join("nvim", "init.work.lua")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, rel := range tt.files {
				mustWrite(t, filepath.Join(root, rel), "profile file\n")
			}

			files, err := Discover(root, []string{"work"}, []string{"work"}, tt.ignore)
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}
			got := relPaths(files)
			assertStringSlicesEqual(t, got, tt.wantRelPath)
		})
	}
}

func TestDiscoverInvalidIgnorePatternErrors(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "nvim", "init.work.lua"), "profile file\n")

	_, err := Discover(root, []string{"work"}, nil, []string{"["})
	if err == nil {
		t.Fatalf("Discover() error = nil, want invalid pattern error")
	}
	if !strings.Contains(err.Error(), "invalid ignore pattern") {
		t.Fatalf("Discover() error = %q, want invalid ignore pattern", err)
	}
}

func relPaths(files []File) []string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.RelPath
	}
	return paths
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
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
