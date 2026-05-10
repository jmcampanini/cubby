package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectDiscoversHostRootAndUnionsDeclaredProfiles(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustMkdir(t, filepath.Join(host, "nested", "dir"))
	mustMkdir(t, src1)
	mustMkdir(t, src2)

	mustWrite(t, filepath.Join(src1, SourceConfigFileName), "profiles = [\"work\", \"client\"]\n")
	mustWrite(t, filepath.Join(src2, SourceConfigFileName), "profiles = [\"work\", \"home\"]\n")
	mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\nname = \"one\"\npath = \"../src1\"\nprofiles = [\"work\"]\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\nignore_conflicts = true\n")

	project, err := LoadProject(filepath.Join(host, "nested", "dir"))
	if err != nil {
		t.Fatalf("LoadProject() error = %v", err)
	}
	if project.HostRoot != host {
		t.Fatalf("HostRoot = %q, want %q", project.HostRoot, host)
	}

	got := project.DeclaredProfiles()
	want := []string{"client", "home", "work"}
	assertStringSlicesEqual(t, got, want)
	if project.Sources[0].ResolvedPath != src1 {
		t.Fatalf("first source path = %q, want %q", project.Sources[0].ResolvedPath, src1)
	}
	if project.Sources[0].IgnoreConflicts {
		t.Fatalf("first source IgnoreConflicts = true, want default false")
	}
	if !project.Sources[1].IgnoreConflicts {
		t.Fatalf("second source IgnoreConflicts = false, want parsed true")
	}
}

func TestResolveSourcePathExpandsHomeAndRelativePaths(t *testing.T) {
	host := filepath.Join(t.TempDir(), "host")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative", path: "../src", want: filepath.Clean(filepath.Join(host, "../src"))},
		{name: "absolute", path: "/tmp/src", want: filepath.Clean("/tmp/src")},
		{name: "home", path: "~/src", want: filepath.Join(home, "src")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSourcePath(host, tt.path)
			if err != nil {
				t.Fatalf("ResolveSourcePath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveSourcePath() = %q, want %q", got, tt.want)
			}
		})
	}
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

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
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
