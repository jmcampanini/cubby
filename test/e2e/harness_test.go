package e2e_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type runResult struct {
	stdout string
	stderr string
	code   int
}

var acceptanceBin string

func TestMain(m *testing.M) {
	root, err := findRepoRoot()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "cubby-e2e-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create e2e build dir: %v\n", err)
		os.Exit(2)
	}
	bin := filepath.Join(tmp, "cubby")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("make", "build", "BIN="+bin)
	cmd.Dir = root
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "make build error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		_ = os.RemoveAll(tmp)
		os.Exit(2)
	}
	acceptanceBin = bin
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

func buildCubby(t *testing.T) string {
	t.Helper()
	if acceptanceBin == "" {
		t.Fatal("acceptance binary was not built")
	}
	return acceptanceBin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Getwd() error = %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat go.mod in %q error = %w", dir, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root from %q", dir)
		}
		dir = parent
	}
}

func runCubby(t *testing.T, bin, dir string, args ...string) runResult {
	t.Helper()
	return runCubbyEnv(t, bin, dir, nil, args...)
}

func runCubbyEnv(t *testing.T, bin, dir string, env map[string]string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = cubbyTestEnv(env)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := runResult{stdout: stdout.String(), stderr: stderr.String()}
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.code = exitErr.ExitCode()
		return result
	}
	t.Fatalf("run cubby %v error = %v", args, err)
	return result
}

func cubbyTestEnv(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "CUBBY_PROFILE=") {
			continue
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func assertFailureContains(t *testing.T, result runResult, want string) {
	t.Helper()
	if result.code == 0 {
		t.Fatalf("exit code = 0, want failure; stdout = %s", result.stdout)
	}
	combined := result.stdout + result.stderr
	assertContains(t, combined, want)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output %q does not contain %q", got, want)
	}
}

func requireCaseSensitiveFilesystem(t *testing.T, dir string) {
	t.Helper()
	probe := filepath.Join(dir, "case-probe")
	mustWrite(t, probe, "probe\n")
	if _, err := os.Lstat(filepath.Join(dir, "CASE-PROBE")); err == nil {
		t.Skip("filesystem is case-insensitive; exact missing-path case-variant behavior is not observable")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Lstat case-sensitivity probe error = %v", err)
	}
}

func assertSymlinkExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
}

func assertSymlinkResolvesTo(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	assertSymlinkExists(t, linkPath)
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", linkPath, err)
	}
	want, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", targetPath, err)
	}
	if resolved != want {
		t.Fatalf("%q resolves to %q, want %q", linkPath, resolved, want)
	}
}

func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", path, err)
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

func mustSymlink(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(linkPath), err)
	}
	target, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		t.Fatalf("Rel(%q, %q) error = %v", filepath.Dir(linkPath), targetPath, err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", target, linkPath, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
