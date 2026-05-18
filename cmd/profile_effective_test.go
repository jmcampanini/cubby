package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestProfileEffectivePrintsHostProfiles(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	unsetEnv(t, "CUBBY_TEST_EXTRA")

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

func TestProfileEffectiveAppendsEnvProfiles(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	t.Setenv("CUBBY_TEST_EXTRA", "personal,work")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "work\npersonal\n" {
		t.Fatalf("stdout = %q, want %q", out, "work\npersonal\n")
	}
}

func TestProfileEffectiveFlagReplacesThenEnvProfilesAppends(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal", "client"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	t.Setenv("CUBBY_TEST_EXTRA", "personal")

	out, errOut, err := executeForTest("profile", "effective", "--profile", "client")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "client\npersonal\n" {
		t.Fatalf("stdout = %q, want %q", out, "client\npersonal\n")
	}
}

func TestProfileEffectiveEmptyWarnsAndExitsZero(t *testing.T) {
	host := writeProfileEffectiveProject(t, "", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	unsetEnv(t, "CUBBY_TEST_EXTRA")

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
	for _, hint := range []string{".cubby.toml", "CUBBY_PROFILE", "--profile", "env_profiles"} {
		if !strings.Contains(errOut, hint) {
			t.Fatalf("stderr = %q, want hint %q", errOut, hint)
		}
	}
}

func TestProfileEffectiveJSONEnvelope(t *testing.T) {
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work", "personal"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	t.Setenv("CUBBY_TEST_EXTRA", "personal")

	out, errOut, err := executeForTest("profile", "effective", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	want := "{\"profiles\":[\"work\",\"personal\"]}\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestProfileEffectiveJSONEmptyIsEmptyArray(t *testing.T) {
	host := writeProfileEffectiveProject(t, "", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	unsetEnv(t, "CUBBY_TEST_EXTRA")

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
	host := writeProfileEffectiveProject(t, "profiles = [\"work\"]\nenv_profiles = \"CUBBY_TEST_EXTRA\"\n", []string{"work"})
	mustChdir(t, host)
	unsetEnv(t, "CUBBY_PROFILE")
	t.Setenv("CUBBY_TEST_EXTRA", "ghost")

	out, errOut, err := executeForTest("profile", "effective")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if out != "work\nghost\n" {
		t.Fatalf("stdout = %q, want %q (undeclared profile should still appear)", out, "work\nghost\n")
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
