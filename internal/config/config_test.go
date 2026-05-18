package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectUsesCurrentDirectoryAsHostRootAndUnionsDeclaredProfiles(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustMkdir(t, host)
	mustMkdir(t, src1)
	mustMkdir(t, src2)

	mustWrite(t, filepath.Join(src1, SourceConfigFileName), "profiles = [\" work \", \"client\", \"work\"]\n")
	mustWrite(t, filepath.Join(src2, SourceConfigFileName), "profiles = [\"work\", \"home\"]\n")
	mustWrite(t, filepath.Join(host, HostConfigFileName), "profiles = [\" work \", \"personal\", \"work\", \"\"]\nignore_conflicts = true\ncase_sensitive = true\n\n[[source]]\nname = \"one\"\npath = \"../src1\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	mustChdir(t, host)

	project, err := LoadProject()
	if err != nil {
		t.Fatalf("LoadProject() error = %v", err)
	}
	wantHost := realPath(t, host)
	if project.HostRoot != wantHost {
		t.Fatalf("HostRoot = %q, want %q", project.HostRoot, wantHost)
	}

	assertStringSlicesEqual(t, project.Host.Profiles, []string{"work", "personal"})
	got := project.DeclaredProfiles()
	want := []string{"client", "home", "work"}
	assertStringSlicesEqual(t, got, want)
	wantSrc1 := realPath(t, src1)
	if project.Sources[0].ResolvedPath != wantSrc1 {
		t.Fatalf("first source path = %q, want %q", project.Sources[0].ResolvedPath, wantSrc1)
	}
	if !project.Host.IgnoreConflicts {
		t.Fatalf("host IgnoreConflicts = false, want parsed true")
	}
	if !project.Host.CaseSensitive {
		t.Fatalf("host CaseSensitive = false, want parsed true")
	}
}

func TestLoadProjectRejectsPerSourceIgnoreConflicts(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustMkdir(t, host)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, SourceConfigFileName), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\nname = \"work\"\npath = \"../src\"\nignore_conflicts = true\n")
	mustChdir(t, host)

	_, err := LoadProject()
	if err == nil {
		t.Fatalf("LoadProject() error = nil, want unknown per-source ignore_conflicts error")
	}
	if !strings.Contains(err.Error(), "unknown keys") || !strings.Contains(err.Error(), "ignore_conflicts") {
		t.Fatalf("LoadProject() error = %q, want strict unknown ignore_conflicts error", err)
	}
}

func TestLoadProjectRejectsPerSourceCaseSensitive(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustMkdir(t, host)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, SourceConfigFileName), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\nname = \"work\"\npath = \"../src\"\ncase_sensitive = true\n")
	mustChdir(t, host)

	_, err := LoadProject()
	if err == nil {
		t.Fatalf("LoadProject() error = nil, want unknown per-source case_sensitive error")
	}
	if !strings.Contains(err.Error(), "unknown keys") || !strings.Contains(err.Error(), "case_sensitive") {
		t.Fatalf("LoadProject() error = %q, want strict unknown case_sensitive error", err)
	}
}

func TestLoadProjectRejectsOldPerSourceHostProfiles(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustMkdir(t, host)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, SourceConfigFileName), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\nname = \"work\"\npath = \"../src\"\nprofiles = [\"work\"]\n")
	mustChdir(t, host)

	_, err := LoadProject()
	if err == nil {
		t.Fatalf("LoadProject() error = nil, want unknown per-source profiles error")
	}
	if !strings.Contains(err.Error(), "unknown keys") || !strings.Contains(err.Error(), "profiles") {
		t.Fatalf("LoadProject() error = %q, want strict unknown profiles error", err)
	}
}

func TestLoadProjectRejectsMissingBlankInvalidAndDuplicateSourceNames(t *testing.T) {
	tests := []struct {
		name       string
		sourceTOML string
		want       string
	}{
		{name: "missing", sourceTOML: "path = \"../src\"", want: "missing name"},
		{name: "blank", sourceTOML: "name = \"   \"\npath = \"../src\"", want: "missing name"},
		{name: "space", sourceTOML: "name = \"bad name\"\npath = \"../src\"", want: "invalid"},
		{name: "dot", sourceTOML: "name = \"bad.name\"\npath = \"../src\"", want: "invalid"},
		{name: "slash", sourceTOML: "name = \"bad/name\"\npath = \"../src\"", want: "invalid"},
		{name: "punctuation", sourceTOML: "name = \"bad!name\"\npath = \"../src\"", want: "invalid"},
		{name: "duplicate", sourceTOML: "name = \"one\"\npath = \"../src\"\n\n[[source]]\nname = \"one\"\npath = \"../src\"", want: "duplicate source name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			host := filepath.Join(tmp, "host")
			src := filepath.Join(tmp, "src")
			mustMkdir(t, host)
			mustMkdir(t, src)
			mustWrite(t, filepath.Join(src, SourceConfigFileName), "profiles = [\"work\"]\n")
			mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\n"+tt.sourceTOML+"\n")
			mustChdir(t, host)

			_, err := LoadProject()
			if err == nil {
				t.Fatalf("LoadProject() error = nil, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadProject() error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadProjectRequiresHostConfigInCurrentDirectory(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	nested := filepath.Join(host, "nested")
	mustMkdir(t, nested)
	mustWrite(t, filepath.Join(host, HostConfigFileName), "[[source]]\nname = \"work\"\npath = \"../src\"\n")
	mustChdir(t, nested)

	_, err := LoadProject()
	if err == nil {
		t.Fatalf("LoadProject() error = nil, want missing current-directory host config error")
	}
	if !strings.Contains(err.Error(), "current directory") || !strings.Contains(err.Error(), "host repo root") {
		t.Fatalf("LoadProject() error = %q, want root-only guidance", err)
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

func TestNormalizeHostConfigTrimsEnvProfiles(t *testing.T) {
	cfg := NormalizeHostConfig(HostConfig{EnvProfiles: "  CUBBY_EXTRA  "})
	if cfg.EnvProfiles != "CUBBY_EXTRA" {
		t.Fatalf("EnvProfiles = %q, want %q", cfg.EnvProfiles, "CUBBY_EXTRA")
	}
	cfg = NormalizeHostConfig(HostConfig{EnvProfiles: "   "})
	if cfg.EnvProfiles != "" {
		t.Fatalf("whitespace-only EnvProfiles = %q, want empty (sentinel)", cfg.EnvProfiles)
	}
}

func TestEffectiveProfiles(t *testing.T) {
	const envKey = "CUBBY_TEST_EFFECTIVE_PROFILES"

	tests := []struct {
		name        string
		profiles    []string
		envProfiles string
		envValue    string
		envUnset    bool
		want        []string
	}{
		{
			name:     "no env_profiles returns normalized raw",
			profiles: []string{"work", " work ", "personal", ""},
			want:     []string{"work", "personal"},
		},
		{
			name:        "env_profiles set but env var unset",
			profiles:    []string{"work"},
			envProfiles: envKey,
			envUnset:    true,
			want:        []string{"work"},
		},
		{
			name:        "env_profiles set, env var empty string",
			profiles:    []string{"work"},
			envProfiles: envKey,
			envValue:    "",
			want:        []string{"work"},
		},
		{
			name:        "env_profiles appends and dedupes first-seen",
			profiles:    []string{"a"},
			envProfiles: envKey,
			envValue:    "a,b",
			want:        []string{"a", "b"},
		},
		{
			name:        "env_profiles trims whitespace and drops blanks",
			profiles:    nil,
			envProfiles: envKey,
			envValue:    " foo , , bar ",
			want:        []string{"foo", "bar"},
		},
		{
			name:        "raw order preserved, env appended after",
			profiles:    []string{"b", "a"},
			envProfiles: envKey,
			envValue:    "c,a",
			want:        []string{"b", "a", "c"},
		},
		{
			name:        "env_profiles whitespace-only env value",
			profiles:    []string{"work"},
			envProfiles: envKey,
			envValue:    "  ,  ,  ",
			want:        []string{"work"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envProfiles != "" {
				if tt.envUnset {
					t.Setenv(envKey, "")
					if err := os.Unsetenv(envKey); err != nil {
						t.Fatalf("Unsetenv(%q) error = %v", envKey, err)
					}
				} else {
					t.Setenv(envKey, tt.envValue)
				}
			}
			cfg := HostConfig{Profiles: tt.profiles, EnvProfiles: tt.envProfiles}
			got := EffectiveProfiles(cfg)
			assertStringSlicesEqual(t, got, tt.want)
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

func realPath(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return real
}

func mustChdir(t *testing.T, path string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatalf("Chdir(%q) error = %v", path, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory %q error = %v", oldWD, err)
		}
	})
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
