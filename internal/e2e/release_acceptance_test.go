package e2e_test

import (
	"strings"
	"testing"
)

func TestReleaseAcceptanceVersionIsInjected(t *testing.T) {
	bin := buildCubby(t)
	result := runCubby(t, bin, repoRoot(t), "--version")
	if result.code != 0 {
		t.Fatalf("cubby --version code = %d, stderr = %s", result.code, result.stderr)
	}
	if !strings.Contains(result.stdout, "cubby version ") {
		t.Fatalf("version output = %q, want cubby version string", result.stdout)
	}
	if strings.Contains(result.stdout, "n/a") {
		t.Fatalf("version output = %q, want injected version", result.stdout)
	}
}

func TestReleaseAcceptanceHelpExposesV01CommandSurface(t *testing.T) {
	bin := buildCubby(t)
	result := runCubby(t, bin, repoRoot(t), "--help")
	if result.code != 0 {
		t.Fatalf("cubby --help code = %d, stderr = %s", result.code, result.stderr)
	}
	for _, want := range []string{"link", "unlink", "prune", "status", "doctor", "profile", "source", "gitignore", "lazygit"} {
		assertContains(t, result.stdout, want)
	}
}
