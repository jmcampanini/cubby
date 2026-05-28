package cmd

import (
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestConfigCommandPrintsEffectiveConfigAndProvenance(t *testing.T) {
	unsetEnv(t, "CUBBY_PROFILE")
	host := writeProfileSelectionProject(t, []string{"work"}, []string{"work"})
	mustChdir(t, host)

	out, errOut, err := executeForTest("config")
	if err != nil {
		t.Fatalf("config error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"profiles = [\"work\"]", "[[source]]", "# Provenance", config.HostConfigFileName} {
		if !strings.Contains(out, want) {
			t.Fatalf("config output missing %q:\n%s", want, out)
		}
	}
}

func TestConfigValidateSupportsHostAndSourceConfig(t *testing.T) {
	host := writeProfileSelectionProject(t, []string{"work"}, []string{"work"})
	mustChdir(t, host)

	out, errOut, err := executeForTest("config", "--validate", config.HostConfigFileName)
	if err != nil {
		t.Fatalf("config --validate host error = %v, stderr = %s", err, errOut)
	}
	if strings.TrimSpace(out) != "valid" {
		t.Fatalf("host validation output = %q, want valid", out)
	}

	out, errOut, err = executeForTest("config", "--validate", "../src/"+config.SourceConfigFileName, "--source-config")
	if err != nil {
		t.Fatalf("config --validate source error = %v, stderr = %s", err, errOut)
	}
	if strings.TrimSpace(out) != "valid" {
		t.Fatalf("source validation output = %q, want valid", out)
	}
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
