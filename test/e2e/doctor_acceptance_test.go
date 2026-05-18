package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDoctorAcceptanceHealthySetupIsQuiet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "ok.work"), "ok\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "/.cubby.toml\n*.work.*\n*.work\n")
	mustSymlink(t, filepath.Join(host, "ok.work"), filepath.Join(src, "ok.work"))

	result := runCubby(t, bin, host, "doctor")
	if result.code != 0 {
		t.Fatalf("doctor code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("doctor stdout = %q, want empty", result.stdout)
	}
}

func TestDoctorFlagsUndeclaredEnvProfilesContribution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nenv_profiles = \"CUBBY_EXTRA\"\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "/.cubby.toml\n*.work.*\n*.work\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_EXTRA": "ghost"}, "doctor")
	if result.code == 0 {
		t.Fatalf("doctor code = 0, want unhealthy; stdout = %s, stderr = %s", result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "MISSING_PROFILE ghost")
}

func TestDoctorAcceptanceUnhealthySetupReportsStableMarkers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	missing := filepath.Join(tmp, "missing")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "ok.work"), "ok\n")
	mustWrite(t, filepath.Join(src, "stale.work"), "stale\n")
	if err := os.Remove(filepath.Join(src, "stale.work")); err != nil {
		t.Fatalf("Remove stale source error = %v", err)
	}
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\", \"ghost\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n\n[[source]]\nname = \"missing\"\npath = \""+missing+"\"\n")
	mustWrite(t, filepath.Join(host, "ok.work"), "conflict\n")
	mustSymlink(t, filepath.Join(host, "stale.work"), filepath.Join(src, "stale.work"))

	result := runCubby(t, bin, host, "doctor")
	if result.code == 0 {
		t.Fatalf("doctor code = 0, want unhealthy failure; stdout = %s", result.stdout)
	}
	for _, want := range []string{
		"MISSING_SOURCE missing",
		"path does not exist",
		"MISSING_GITIGNORE *.work",
		"MISSING_PROFILE ghost",
		"DANGLING stale.work",
		"CONFLICT ok.work",
	} {
		assertContains(t, result.stdout, want)
	}
}
