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
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")
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

func TestLinkConflictSafetyEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(src, "create.work"), "source create\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want conflict failure; stdout = %s", result.stdout)
	}
	assertContains(t, result.stdout, "CONFLICT conflict.work")
	if got := mustRead(t, filepath.Join(host, "conflict.work")); got != "host conflict\n" {
		t.Fatalf("conflict file = %q, want untouched", got)
	}
	assertNotExist(t, filepath.Join(host, "create.work"))
}

func TestLinkUnexpectedSymlinkConflictEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	other := filepath.Join(tmp, "other")
	hostPath := filepath.Join(host, "conflict.work")
	otherPath := filepath.Join(other, "conflict.work")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source\n")
	mustWrite(t, otherPath, "other\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, hostPath, otherPath)

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want unexpected symlink failure")
	}
	assertContains(t, result.stdout, "CONFLICT conflict.work")
	assertSymlinkResolvesTo(t, hostPath, otherPath)
}

func TestLinkIgnoreConflictsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(src, "create.work"), "source create\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	result := runCubby(t, bin, host, "link", "--ignore-conflicts")
	if result.code != 0 {
		t.Fatalf("link --ignore-conflicts code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "SKIP conflict.work")
	assertSymlinkExists(t, filepath.Join(host, "create.work"))
	if got := mustRead(t, filepath.Join(host, "conflict.work")); got != "host conflict\n" {
		t.Fatalf("conflict file = %q, want untouched", got)
	}
}

func TestLinkHostConfigIgnoreConflictsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(src, "create.work"), "source create\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nignore_conflicts = true\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	result := runCubby(t, bin, host, "link")
	if result.code != 0 {
		t.Fatalf("link with host ignore_conflicts code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "SKIP conflict.work")
	assertSymlinkExists(t, filepath.Join(host, "create.work"))
}

func TestLinkCrossSourceCollisionEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src1, "same.work"), "one\n")
	mustWrite(t, filepath.Join(src1, "unique.work"), "unique\n")
	mustWrite(t, filepath.Join(src2, "same.work"), "two\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want collision failure")
	}
	assertContains(t, result.stdout, "CONFLICT same.work")
	assertContains(t, result.stdout, "collision")
	assertNotExist(t, filepath.Join(host, "same.work"))
	assertNotExist(t, filepath.Join(host, "unique.work"))

	ignoredHost := filepath.Join(tmp, "ignored-host")
	mustWrite(t, filepath.Join(ignoredHost, ".cubby.toml"), "profiles = [\"work\"]\nignore_conflicts = true\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")
	ignored := runCubby(t, bin, ignoredHost, "link")
	if ignored.code != 0 {
		t.Fatalf("ignored collision code = %d, stdout = %s, stderr = %s", ignored.code, ignored.stdout, ignored.stderr)
	}
	assertContains(t, ignored.stdout, "SKIP same.work")
	assertSymlinkResolvesTo(t, filepath.Join(ignoredHost, "same.work"), filepath.Join(src1, "same.work"))
	assertSymlinkExists(t, filepath.Join(ignoredHost, "unique.work"))
}

func TestLinkDryRunEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "create.work"), "create\n")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")
	mustSymlink(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))

	result := runCubby(t, bin, host, "link", "--dry-run")
	if result.code == 0 {
		t.Fatalf("link --dry-run code = 0, want conflict-equivalent failure")
	}
	assertContains(t, result.stdout, "CREATE create.work")
	assertContains(t, result.stdout, "NOOP linked.work already linked")
	assertContains(t, result.stdout, "CONFLICT conflict.work")
	assertNotExist(t, filepath.Join(host, "create.work"))
	if got := mustRead(t, filepath.Join(host, "conflict.work")); got != "host conflict\n" {
		t.Fatalf("conflict file = %q, want untouched", got)
	}
}

func TestUnlinkDryRunAndSkipSafetyEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	other := filepath.Join(tmp, "other")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "regular.work"), "regular source\n")
	mustWrite(t, filepath.Join(src, "unexpected.work"), "unexpected source\n")
	mustWrite(t, filepath.Join(src, "missing.work"), "missing source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))
	mustWrite(t, filepath.Join(host, "regular.work"), "host regular\n")
	mustWrite(t, filepath.Join(other, "unexpected.work"), "other\n")
	mustSymlink(t, filepath.Join(host, "unexpected.work"), filepath.Join(other, "unexpected.work"))

	result := runCubby(t, bin, host, "unlink", "--dry-run")
	if result.code != 0 {
		t.Fatalf("unlink --dry-run code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "REMOVE linked.work")
	assertContains(t, result.stdout, "SKIP regular.work")
	assertContains(t, result.stdout, "SKIP unexpected.work")
	assertContains(t, result.stdout, "NOOP missing.work missing")
	assertSymlinkExists(t, filepath.Join(host, "linked.work"))
	if got := mustRead(t, filepath.Join(host, "regular.work")); got != "host regular\n" {
		t.Fatalf("regular file = %q, want untouched", got)
	}
	assertSymlinkResolvesTo(t, filepath.Join(host, "unexpected.work"), filepath.Join(other, "unexpected.work"))
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
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+src+"\"\n")

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

func TestLinkUsesHostDefaultProfiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.personal.lua"), "-- personal\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertNotExist(t, filepath.Join(host, "nvim", "init.personal.lua"))
}

func TestUnlinkUsesHostDefaultProfiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	hostFile := filepath.Join(host, "nvim", "init.work.lua")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	link := runCubby(t, bin, host, "link")
	if link.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", link.code, link.stdout, link.stderr)
	}
	assertSymlinkExists(t, hostFile)
	unlink := runCubby(t, bin, host, "unlink")
	if unlink.code != 0 {
		t.Fatalf("unlink code = %d, stdout = %s, stderr = %s", unlink.code, unlink.stdout, unlink.stderr)
	}
	assertNotExist(t, hostFile)
}

func TestLinkUsesEnvFallbackWhenNoFlagIsPresent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.personal.lua"), "-- personal\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_PROFILE": "personal"}, "link")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.personal.lua"))
}

func TestLinkFlagOverridesEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.personal.lua"), "-- personal\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_PROFILE": "work"}, "link", "--profile", "personal")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.personal.lua"))
}

func TestLinkMultiProfileSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\", \"client\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.personal.lua"), "-- personal\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.client.lua"), "-- client\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link", "--profile", "work, personal", "--profile", "work")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.personal.lua"))
	assertNotExist(t, filepath.Join(host, "nvim", "init.client.lua"))

	unlink := runCubby(t, bin, host, "unlink", "--profile", "work,personal")
	if unlink.code != 0 {
		t.Fatalf("unlink code = %d, stdout = %s, stderr = %s", unlink.code, unlink.stdout, unlink.stderr)
	}
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertNotExist(t, filepath.Join(host, "nvim", "init.personal.lua"))
}

func TestLinkNoSelectionErrors(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link")
	assertFailureContains(t, result, "no profiles selected")
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
}

func TestLinkUnknownProfileErrorsBeforeCreatingLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	workHostFile := filepath.Join(host, "nvim", "init.work.lua")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link", "--profile", "ghost,work")
	assertFailureContains(t, result, "ghost")
	assertNotExist(t, workHostFile)
}

func TestLinkHonorsSourceIgnoreRules(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\nignore = [\"nvim/init.work.lua\", \"**/*.draft.*\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- ignored exact\n")
	mustWrite(t, filepath.Join(src, "nvim", "draft.work.draft.lua"), "-- ignored glob\n")
	mustWrite(t, filepath.Join(src, "nvim", "keep.work.lua"), "-- keep\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertNotExist(t, filepath.Join(host, "nvim", "draft.work.draft.lua"))
	assertSymlinkExists(t, filepath.Join(host, "nvim", "keep.work.lua"))
}

func TestLinkIgnoresUndeclaredLookalikes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.client.lua"), "-- client lookalike\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertNotExist(t, filepath.Join(host, "nvim", "init.client.lua"))
}

func TestProfileListEndToEnd(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"client\", \"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"host-only\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "profile", "list")
	if result.code != 0 {
		t.Fatalf("profile list code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	want := "client\npersonal\nwork\n"
	if result.stdout != want {
		t.Fatalf("profile list stdout = %q, want %q", result.stdout, want)
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
