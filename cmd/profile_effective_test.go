package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestProfileEffectivePrintsHostProfiles(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\n", []string{"work", "personal"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILES")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "work\n" {
		t.Fatalf("stdout = %q, want %q", out, "work\n")
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
}

func TestProfileEffectiveUsesProfilesEnv(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\n", []string{"work", "personal"})
	mustChdir(t, host)
	t.Setenv("CUBBY_PROFILES", "personal,work")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "personal\nwork\n" {
		t.Fatalf("stdout = %q, want %q", out, "personal\nwork\n")
	}
}

func TestProfileEffectiveFlagOverridesProfilesEnv(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\n", []string{"work", "personal", "client"})
	mustChdir(t, host)
	t.Setenv("CUBBY_PROFILES", "personal")

	out, errOut, err := executeForTest("profile", "effective", "--profile", "client")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "client\n" {
		t.Fatalf("stdout = %q, want %q", out, "client\n")
	}
}

func TestProfileEffectiveAppendsEnvProfiles(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal", "client"})
	mustChdir(t, host)
	t.Setenv("CUBBY_PROFILES", "client")
	t.Setenv("CUBBY_TEST_EXTRA", "personal,work")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "client\npersonal\nwork\n" {
		t.Fatalf("stdout = %q, want %q", out, "client\npersonal\nwork\n")
	}
}

func TestProfileEffectiveEmptyWarnsAndExitsZero(t *testing.T) {
	host := writeProfileEffectiveProject(t, "", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILES")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "no profiles selected") {
		t.Fatalf("stderr = %q, want warning containing 'no profiles selected'", errOut)
	}
	for _, hint := range []string{".cubby.toml", "CUBBY_PROFILES", "--profiles/--profile", "env_profiles"} {
		if !strings.Contains(errOut, hint) {
			t.Fatalf("stderr = %q, want hint %q", errOut, hint)
		}
	}
}

func TestProfileEffectiveJSONEnvelope(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\n", []string{"work", "personal"})
	mustChdir(t, host)
	t.Setenv("CUBBY_PROFILES", "personal")

	out, errOut, err := executeForTest("profile", "effective", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	want := "{\"profiles\":[\"personal\"]}\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestProfileEffectiveJSONEmptyIsEmptyArray(t *testing.T) {
	host := writeProfileEffectiveProject(t, "", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILES")

	out, errOut, err := executeForTest("profile", "effective", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty in json mode", errOut)
	}
	want := "{\"profiles\":[]}\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestProfileEffectiveDoesNotValidateAgainstDeclaredSources(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"ghost\"]\n", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILES")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "ghost\n" {
		t.Fatalf("stdout = %q, want %q (undeclared profile should still appear)", out, "ghost\n")
	}
}

func writeProfileEffectiveProject(t *testing.T, hostExtra string, sourceProfiles []string) string {
	t.Helper()
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, config.SourceConfigFileName), "profiles = "+tomlStringArray(sourceProfiles)+"\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), hostExtra+"\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	return host
}
