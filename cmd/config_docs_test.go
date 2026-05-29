package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestConfigCommandPrintsEffectiveConfigAsTOML(t *testing.T) {
	unsetEnv(t, "CUBBY_PROFILE")
	host := writeProfileSelectionProject(t, []string{"work"}, []string{"work"})
	mustChdir(t, host)

	out, errOut, err := executeForTest("config")
	if err != nil {
		t.Fatalf("config error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{
		"profiles = [\"work\"]",
		"[[source]]",
		"# Effective",
		`# loaded_files = ["` + filepath.Join(host, config.HostConfigFileName) + `"]`,
		"# effective_profiles = [\"work\"]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("config output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "# Provenance") {
		t.Fatalf("config output includes provenance by default:\n%s", out)
	}
	parsed := parseHostConfigOutput(t, out)
	if len(parsed.Sources) != 1 || parsed.Sources[0].Name != "src" {
		t.Fatalf("parsed sources = %#v, want one src source", parsed.Sources)
	}
}

func TestConfigCommandPrintsEffectiveProfilesWithEnvProfiles(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	t.Setenv("CUBBY_TEST_EXTRA", "personal,work")

	out, errOut, err := executeForTest("config")
	if err != nil {
		t.Fatalf("config error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{
		`profiles = ["work"]`,
		`env_profiles = "CUBBY_TEST_EXTRA"`,
		`# effective_profiles = ["work", "personal"]`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("config output missing %q:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if line == `profiles = ["work", "personal"]` {
			t.Fatalf("raw TOML should not include env_profiles contribution:\n%s", out)
		}
	}
	parseHostConfigOutput(t, out)
}

func TestConfigCommandPrintsProvenanceWhenRequestedAsTOMLComments(t *testing.T) {
	unsetEnv(t, "CUBBY_PROFILE")
	host := writeProfileSelectionProject(t, []string{"work"}, []string{"work"})
	mustChdir(t, host)

	out, errOut, err := executeForTest("config", "--provenance")
	if err != nil {
		t.Fatalf("config --provenance error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"profiles = [\"work\"]", "[[source]]", "# Provenance", config.HostConfigFileName} {
		if !strings.Contains(out, want) {
			t.Fatalf("config --provenance output missing %q:\n%s", want, out)
		}
	}
	parseHostConfigOutput(t, out)
}

func TestConfigValidateSupportsHostAndSourceConfig(t *testing.T) {
	host := writeProfileSelectionProject(t, []string{"work"}, []string{"work"})
	mustChdir(t, t.TempDir())

	out, errOut, err := executeForTest("config", "--validate", filepath.Join(host, config.HostConfigFileName))
	if err != nil {
		t.Fatalf("config --validate host error = %v, stderr = %s", err, errOut)
	}
	if strings.TrimSpace(out) != "valid" {
		t.Fatalf("host validation output = %q, want valid", out)
	}

	sourceFile := filepath.Join(filepath.Dir(host), "src", config.SourceConfigFileName)
	out, errOut, err = executeForTest("config", "--validate", sourceFile, "--source-config")
	if err != nil {
		t.Fatalf("config --validate source error = %v, stderr = %s", err, errOut)
	}
	if strings.TrimSpace(out) != "valid" {
		t.Fatalf("source validation output = %q, want valid", out)
	}
}

func TestConfigValidateRejectsInvalidHostSourceSemantics(t *testing.T) {
	tests := []struct {
		name     string
		hostTOML string
		want     string
	}{
		{name: "no sources", hostTOML: "profiles = [\"work\"]\n", want: "no [[source]] entries"},
		{name: "duplicate source names", hostTOML: "[[source]]\nname = \"src\"\npath = \"../src\"\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n", want: "duplicate source name"},
		{name: "missing source path", hostTOML: "[[source]]\nname = \"src\"\n", want: "missing path"},
		{name: "nonexistent source path", hostTOML: "[[source]]\nname = \"src\"\npath = \"../missing\"\n", want: "path does not exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			host := filepath.Join(root, "host")
			src := filepath.Join(root, "src")
			mustWrite(t, filepath.Join(src, config.SourceConfigFileName), "profiles = [\"work\"]\n")
			mustWrite(t, filepath.Join(host, config.HostConfigFileName), tt.hostTOML)

			_, _, err := executeForTest("config", "--validate", filepath.Join(host, config.HostConfigFileName))
			if err == nil {
				t.Fatalf("config --validate error = nil, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("config --validate error = %q, want %q", err, tt.want)
			}
		})
	}
}

func parseHostConfigOutput(t *testing.T, out string) config.HostConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), config.HostConfigFileName)
	mustWrite(t, path, out)
	parsed, err := config.LoadHostConfigFile(path)
	if err != nil {
		t.Fatalf("config output is not reusable TOML: %v\n%s", err, out)
	}
	return parsed
}

func TestDocsCommandPrintsSchema(t *testing.T) {
	out, errOut, err := executeForTest("docs", "schema")
	if err != nil {
		t.Fatalf("docs schema error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"# Cubby config schema", config.HostConfigFileName, config.SourceConfigFileName} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs schema output missing %q:\n%s", want, out)
		}
	}
}
