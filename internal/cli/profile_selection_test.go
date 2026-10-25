package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestLoadProfileScopedProjectSelectionPrecedence(t *testing.T) {
	tests := []struct {
		name         string
		hostProfiles []string
		envProfiles  *string
		args         []string
		wantProfiles []string
		wantErr      string
	}{
		{
			name:         "host defaults used when no flag or env is present",
			hostProfiles: []string{"work", "personal", "work"},
			wantProfiles: []string{"work", "personal"},
		},
		{
			name:         "CUBBY_PROFILES falls back over host defaults",
			hostProfiles: []string{"work"},
			envProfiles:  stringPtr("personal"),
			wantProfiles: []string{"personal"},
		},
		{
			name:         "CSV CUBBY_PROFILES input is split and trimmed",
			hostProfiles: []string{"client"},
			envProfiles:  stringPtr("work, personal"),
			wantProfiles: []string{"work", "personal"},
		},
		{
			name:         "CUBBY_PROFILES keeps last occurrences and drops blanks",
			hostProfiles: []string{"client"},
			envProfiles:  stringPtr("work, personal, work, ,client,personal"),
			wantProfiles: []string{"work", "client", "personal"},
		},
		{
			name:         "flag overrides env completely",
			hostProfiles: []string{"work"},
			envProfiles:  stringPtr("work"),
			args:         []string{"--profile", "personal"},
			wantProfiles: []string{"personal"},
		},
		{
			name:         "repeated singular flags keep last occurrences",
			args:         []string{"--profile", "work", "--profile", "personal", "--profile", "work"},
			wantProfiles: []string{"personal", "work"},
		},
		{
			name:         "plural CSV flag input is split and trimmed",
			args:         []string{"--profiles", "work, personal"},
			wantProfiles: []string{"work", "personal"},
		},
		{
			name:         "repeated plural CSV flags keep last occurrences and drop blanks",
			args:         []string{"--profiles", "work,personal,work", "--profiles", " ,client,personal"},
			wantProfiles: []string{"work", "client", "personal"},
		},
		{
			name:         "mixed plural CSV and singular flags keep last occurrences after combining",
			args:         []string{"--profiles", "work,personal", "--profile", "work", "--profile", "client"},
			wantProfiles: []string{"personal", "work", "client"},
		},
		{
			name:    "empty effective selection errors",
			wantErr: "no profiles selected",
		},
		{
			name:         "changed empty plural flag errors instead of falling back to env or host defaults",
			hostProfiles: []string{"work"},
			envProfiles:  stringPtr("personal"),
			args:         []string{"--profiles="},
			wantErr:      "no profiles selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := writeProfileSelectionProject(t, tt.hostProfiles, []string{"work", "personal", "client"})
			mustChdir(t, host)
			if tt.envProfiles == nil {
				unsetEnv(t, "CUBBY_PROFILES")
			} else {
				t.Setenv("CUBBY_PROFILES", *tt.envProfiles)
			}

			cmd := linkCommand(commandIO{})
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}
			_, profiles, err := loadProfileScopedProject(cmd)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("loadProfileScopedProject() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadProfileScopedProject() error = %q, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadProfileScopedProject() error = %v", err)
			}
			assertProfilesEqual(t, profiles, tt.wantProfiles)
		})
	}
}

func TestProfileSingularFlagRejectsEmpty(t *testing.T) {
	cmd := linkCommand(commandIO{})
	if err := cmd.ParseFlags([]string{"--profile="}); err == nil {
		t.Fatal("ParseFlags() error = nil for empty singular --profile")
	}
}

func TestValidateSelectedProfilesRejectsUnknownBeforeSideEffects(t *testing.T) {
	project := &config.Project{
		Sources: []config.RegisteredSource{
			{Config: config.SourceConfig{Profiles: []string{"work", "personal"}}},
		},
	}

	err := config.ValidateSelectedProfiles(project, []string{"work", "ghost"})
	if err == nil {
		t.Fatalf("config.ValidateSelectedProfiles() error = nil, want unknown profile")
	}
	if !strings.Contains(err.Error(), "ghost") || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("config.ValidateSelectedProfiles() error = %q, want unknown profile guidance", err)
	}
}

func TestRenderMissingProfileDiagnosticsReportsPerSourceMissingProfiles(t *testing.T) {
	project := &config.Project{Sources: []config.RegisteredSource{
		{HostSource: config.HostSource{Name: "personal"}, Config: config.SourceConfig{Profiles: []string{"personal"}}},
		{HostSource: config.HostSource{Name: "work"}, Config: config.SourceConfig{Profiles: []string{"work"}}},
	}}
	var stderr bytes.Buffer
	cmd := NewRootCommand(&bytes.Buffer{}, &stderr)

	if err := renderMissingProfileDiagnostics(cmd, project, []string{"work", "personal"}); err != nil {
		t.Fatalf("renderMissingProfileDiagnostics() error = %v", err)
	}
	want := "source \"personal\" does not declare selected profile \"work\"; skipping\n" +
		"source \"work\" does not declare selected profile \"personal\"; skipping\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestSourceSelectedProfilesUsesSourceDeclaredProfiles(t *testing.T) {
	source := config.RegisteredSource{Config: config.SourceConfig{Profiles: []string{"work", "client"}}}
	got := config.SourceSelectedProfiles(source, []string{"home", "work", "client", "work"})
	assertProfilesEqual(t, got, []string{"work", "client"})
}

func TestSourceSelectedProfilesOmittedSourceProfilesSelectNothing(t *testing.T) {
	source := config.RegisteredSource{}
	got := config.SourceSelectedProfiles(source, []string{"work"})
	assertProfilesEqual(t, got, nil)
}

func writeProfileSelectionProject(t *testing.T, hostProfiles, sourceProfiles []string) string {
	t.Helper()
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, config.SourceConfigFileName), "profiles = "+tomlStringArray(sourceProfiles)+"\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), hostProfilesTOML(hostProfiles)+"[[source]]\nname = \"src\"\npath = \"../src\"\n")
	return host
}

func hostProfilesTOML(profiles []string) string {
	if profiles == nil {
		return ""
	}
	return "profiles = " + tomlStringArray(profiles) + "\n\n"
}

func tomlStringArray(values []string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = `"` + value + `"`
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func stringPtr(value string) *string {
	return &value
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", key, err)
	}
	t.Cleanup(func() {
		if ok {
			if err := os.Setenv(key, old); err != nil {
				t.Fatalf("restore env %s error = %v", key, err)
			}
			return
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("clear env %s error = %v", key, err)
		}
	})
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

func assertProfilesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("profiles = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("profiles = %#v, want %#v", got, want)
		}
	}
}
