package e2e_test

import (
	"bytes"
	"errors"
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

func TestLinkUnlinkSingleSourceSmoke(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	sourceFile := filepath.Join(src, "nvim", "init.work.lua")
	hostFile := filepath.Join(host, "nvim", "init.work.lua")
	mustMkdir(t, host)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\nprofiles = [\"work\"]\n")
	mustWrite(t, sourceFile, "-- work\n")

	link := runCubby(t, bin, host, "link", "--profile", "work")
	if link.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", link.code, link.stdout, link.stderr)
	}
	info, err := os.Lstat(hostFile)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", hostFile, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", hostFile, info.Mode())
	}
	target, err := os.Readlink(hostFile)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", hostFile, err)
	}
	if filepath.IsAbs(target) {
		t.Fatalf("symlink target = %q, want relative", target)
	}
	resolved, err := filepath.EvalSymlinks(hostFile)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", hostFile, err)
	}
	wantResolved, err := filepath.EvalSymlinks(sourceFile)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", sourceFile, err)
	}
	if resolved != wantResolved {
		t.Fatalf("resolved symlink = %q, want %q", resolved, wantResolved)
	}

	linkAgain := runCubby(t, bin, host, "link", "--profile", "work")
	if linkAgain.code != 0 {
		t.Fatalf("second link code = %d, stdout = %s, stderr = %s", linkAgain.code, linkAgain.stdout, linkAgain.stderr)
	}

	checkBefore := runCubby(t, bin, host, "gitignore", "check")
	if checkBefore.code == 0 {
		t.Fatalf("gitignore check before sync code = 0, stdout = %s", checkBefore.stdout)
	}
	assertContains(t, checkBefore.stdout, "*.work.*")
	assertContains(t, checkBefore.stdout, "*.work")
	sync := runCubby(t, bin, host, "gitignore", "sync")
	if sync.code != 0 {
		t.Fatalf("gitignore sync code = %d, stderr = %s", sync.code, sync.stderr)
	}
	checkAfter := runCubby(t, bin, host, "gitignore", "check")
	if checkAfter.code != 0 {
		t.Fatalf("gitignore check after sync code = %d, stdout = %s, stderr = %s", checkAfter.code, checkAfter.stdout, checkAfter.stderr)
	}

	unlink := runCubby(t, bin, host, "unlink", "--profile", "work")
	if unlink.code != 0 {
		t.Fatalf("unlink code = %d, stdout = %s, stderr = %s", unlink.code, unlink.stdout, unlink.stderr)
	}
	if _, err := os.Lstat(hostFile); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", hostFile, err)
	}

	mustWrite(t, hostFile, "host-owned\n")
	unlinkRegular := runCubby(t, bin, host, "unlink", "--profile", "work")
	if unlinkRegular.code != 0 {
		t.Fatalf("unlink regular code = %d, stdout = %s, stderr = %s", unlinkRegular.code, unlinkRegular.stdout, unlinkRegular.stderr)
	}
	if got := mustRead(t, hostFile); got != "host-owned\n" {
		t.Fatalf("regular host file content = %q, want untouched", got)
	}
}

func TestGitignoreCheckSyncEndToEndFromHostRoot(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	nested := filepath.Join(host, "nested", "dir")
	mustMkdir(t, nested)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\nprofiles = [\"work\"]\n")

	checkBefore := runCubby(t, bin, host, "gitignore", "check")
	if checkBefore.code == 0 {
		t.Fatalf("gitignore check before sync code = 0, stdout = %s", checkBefore.stdout)
	}
	assertContains(t, checkBefore.stdout, "*.work.*")
	assertContains(t, checkBefore.stdout, "*.work")
	if checkBefore.stderr != "" {
		t.Fatalf("gitignore check before sync stderr = %q, want empty", checkBefore.stderr)
	}

	sync := runCubby(t, bin, host, "gitignore", "sync")
	if sync.code != 0 {
		t.Fatalf("gitignore sync code = %d, stderr = %s", sync.code, sync.stderr)
	}
	assertContains(t, sync.stdout, "*.work.*")
	assertContains(t, sync.stdout, "*.work")

	rootGitignore := filepath.Join(host, ".gitignore")
	content := mustRead(t, rootGitignore)
	assertContains(t, content, "*.work.*\n")
	assertContains(t, content, "*.work\n")
	if _, err := os.Stat(filepath.Join(nested, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("nested .gitignore stat error = %v, want not exist", err)
	}

	checkAfter := runCubby(t, bin, host, "gitignore", "check")
	if checkAfter.code != 0 {
		t.Fatalf("gitignore check after sync code = %d, stdout = %s, stderr = %s", checkAfter.code, checkAfter.stdout, checkAfter.stderr)
	}
	if checkAfter.stdout != "" {
		t.Fatalf("gitignore check after sync stdout = %q, want empty", checkAfter.stdout)
	}

	syncAgain := runCubby(t, bin, host, "gitignore", "sync")
	if syncAgain.code != 0 {
		t.Fatalf("second gitignore sync code = %d, stderr = %s", syncAgain.code, syncAgain.stderr)
	}
	if syncAgain.stdout != "" {
		t.Fatalf("second gitignore sync stdout = %q, want empty", syncAgain.stdout)
	}
	content = mustRead(t, rootGitignore)
	if got := strings.Count(content, "*.work.*"); got != 1 {
		t.Fatalf("*.work.* count = %d, want 1 in %q", got, content)
	}
	if got := strings.Count(content, "*.work\n"); got != 1 {
		t.Fatalf("*.work count = %d, want 1 in %q", got, content)
	}

	nestedResult := runCubby(t, bin, nested, "gitignore", "sync")
	if nestedResult.code == 0 {
		t.Fatalf("nested gitignore sync code = 0, want failure")
	}
	assertContains(t, nestedResult.stderr, "run cubby from the host repo root")
	if _, err := os.Stat(filepath.Join(nested, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("nested .gitignore stat error = %v, want not exist", err)
	}
}

func TestGitignoreSyncAppendsToExistingGitignoreReadably(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustMkdir(t, host)
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "existing")

	result := runCubby(t, bin, host, "gitignore", "sync")
	if result.code != 0 {
		t.Fatalf("gitignore sync code = %d, stderr = %s", result.code, result.stderr)
	}
	want := "existing\n*.work.*\n*.work\n"
	if got := mustRead(t, filepath.Join(host, ".gitignore")); got != want {
		t.Fatalf(".gitignore = %q, want %q", got, want)
	}
}

func TestConfigErrorCases(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()

	t.Run("missing host config", func(t *testing.T) {
		dir := filepath.Join(tmp, "missing-host")
		mustMkdir(t, dir)
		result := runCubby(t, bin, dir, "gitignore", "check")
		assertFailureContains(t, result, ".cubby.toml")
	})

	t.Run("missing source path", func(t *testing.T) {
		host := filepath.Join(tmp, "missing-source-path")
		mustMkdir(t, host)
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \"./does-not-exist\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "path does not exist")
	})

	t.Run("missing source config", func(t *testing.T) {
		host := filepath.Join(tmp, "missing-source-config-host")
		src := filepath.Join(tmp, "missing-source-config-src")
		mustMkdir(t, host)
		mustMkdir(t, src)
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "cubby.toml")
	})

	t.Run("source config with no declared profiles", func(t *testing.T) {
		host := filepath.Join(tmp, "no-profiles-host")
		src := filepath.Join(tmp, "no-profiles-src")
		mustMkdir(t, host)
		mustMkdir(t, src)
		mustWrite(t, filepath.Join(src, "cubby.toml"), "ignore = []\n")
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
		result := runCubby(t, bin, host, "gitignore", "check")
		assertFailureContains(t, result, "declares no profiles")
	})
}

func buildCubby(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "cubby")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = root
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat go.mod in %q error = %v", dir, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %q", dir)
		}
		dir = parent
	}
}

func runCubby(t *testing.T, bin, dir string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
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

func mustRead(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
