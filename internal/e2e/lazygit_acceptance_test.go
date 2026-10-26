package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Source selection and the PATH lookup are proven at unit level with the fake
// runner and the fake terminal check. Through the binary, piped stdio is the
// observable case: the terminal check fails before lazygit starts.
func TestLazygitAcceptanceWithoutTerminalFailsBeforeStarting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script execution is not portable enough for this acceptance test")
	}
	bin := buildCubby(t)
	host, _ := writeLazygitAcceptanceProject(t, "one")
	fakeDir, logPath := writeFakeLazygit(t)

	result := runCubbyEnv(t, bin, host, map[string]string{"PATH": fakeDir, "CUBBY_LAZYGIT_LOG": logPath}, "lazygit")

	if result.code != 1 {
		t.Fatalf("lazygit code = %d, want 1; stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("lazygit stdout = %q, want empty", result.stdout)
	}
	if !strings.HasPrefix(result.stderr, "cubby: lazygit needs a terminal") || strings.Count(result.stderr, "\n") != 1 {
		t.Fatalf("lazygit stderr = %q, want one line of terminal guidance", result.stderr)
	}
	assertNotExist(t, logPath)
}

func TestLazygitAcceptanceMultipleSourcesWithoutSourceFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script execution is not portable enough for this acceptance test")
	}
	bin := buildCubby(t)
	host, _ := writeLazygitAcceptanceProject(t, "one", "two")
	fakeDir, logPath := writeFakeLazygit(t)

	result := runCubbyEnv(t, bin, host, map[string]string{"PATH": fakeDir, "CUBBY_LAZYGIT_LOG": logPath}, "lazygit")
	assertFailureContains(t, result, "multiple sources")
	assertContains(t, result.stderr, "--source")
	assertNotExist(t, logPath)
}

func TestLazygitAcceptanceUnknownSourceFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script execution is not portable enough for this acceptance test")
	}
	bin := buildCubby(t)
	host, _ := writeLazygitAcceptanceProject(t, "one", "two")
	fakeDir, logPath := writeFakeLazygit(t)

	result := runCubbyEnv(t, bin, host, map[string]string{"PATH": fakeDir, "CUBBY_LAZYGIT_LOG": logPath}, "lazygit", "--source", "missing")
	assertFailureContains(t, result, "unknown source \"missing\"")
	assertContains(t, result.stderr, "one")
	assertContains(t, result.stderr, "two")
	assertNotExist(t, logPath)
}

func TestLazygitAcceptanceMissingBinaryFailsClearly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script execution is not portable enough for this acceptance test")
	}
	bin := buildCubby(t)
	host, _ := writeLazygitAcceptanceProject(t, "one")
	emptyPath := t.TempDir()

	result := runCubbyEnv(t, bin, host, map[string]string{"PATH": emptyPath}, "lazygit")
	assertFailureContains(t, result, "lazygit not found in PATH")
}

func writeLazygitAcceptanceProject(t *testing.T, sourceNames ...string) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	host := filepath.Join(root, "host")
	sources := make(map[string]string, len(sourceNames))
	hostConfig := ""
	for _, name := range sourceNames {
		sourcePath := filepath.Join(root, name)
		sources[name] = filepath.Clean(sourcePath)
		mustWrite(t, filepath.Join(sourcePath, "cubby.toml"), "profiles = [\"work\"]\n")
		hostConfig += "[[source]]\nname = \"" + name + "\"\npath = \"" + sourcePath + "\"\n\n"
	}
	mustWrite(t, filepath.Join(host, ".cubby.toml"), hostConfig)
	return host, sources
}

func writeFakeLazygit(t *testing.T) (dir, logPath string) {
	t.Helper()
	dir = t.TempDir()
	logPath = filepath.Join(t.TempDir(), "lazygit.log")
	path := filepath.Join(dir, "lazygit")
	content := "#!/bin/sh\npwd -P > \"$CUBBY_LAZYGIT_LOG\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return dir, logPath
}
