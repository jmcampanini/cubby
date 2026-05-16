package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

func TestLinkCaseCollisionDefaultEndToEnd(t *testing.T) {
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
	mustWrite(t, filepath.Join(src1, "foo.work"), "one\n")
	mustWrite(t, filepath.Join(src2, "FOO.work"), "two\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want case collision failure")
	}
	assertContains(t, result.stdout, "CONFLICT FOO.work")
	assertContains(t, result.stdout, "path case collision with foo.work")
	assertNotExist(t, filepath.Join(host, "foo.work"))
}

func TestLinkCaseCollisionDryRunEndToEnd(t *testing.T) {
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
	mustWrite(t, filepath.Join(src1, "foo.work"), "one\n")
	mustWrite(t, filepath.Join(src2, "FOO.work"), "two\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "link", "--dry-run")
	if result.code == 0 {
		t.Fatalf("link --dry-run code = 0, want case collision failure")
	}
	assertContains(t, result.stdout, "CREATE foo.work")
	assertContains(t, result.stdout, "CONFLICT FOO.work")
	assertContains(t, result.stdout, "path case collision")
	assertNotExist(t, filepath.Join(host, "foo.work"))
}

func TestLinkCaseCollisionIgnoreConflictsEndToEnd(t *testing.T) {
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
	mustWrite(t, filepath.Join(src1, "foo.work"), "one\n")
	mustWrite(t, filepath.Join(src2, "FOO.work"), "two\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "link", "--ignore-conflicts")
	if result.code != 0 {
		t.Fatalf("link --ignore-conflicts code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "SKIP FOO.work")
	assertContains(t, result.stdout, "path case collision")
	assertSymlinkResolvesTo(t, filepath.Join(host, "foo.work"), filepath.Join(src1, "foo.work"))
}

func TestLinkExistingHostCaseConflictDefaultEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	requireCaseSensitiveFilesystem(t, tmp)
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "FOO.work"), "source\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want host case conflict failure")
	}
	assertContains(t, result.stdout, "CONFLICT FOO.work")
	assertContains(t, result.stdout, "host path case conflict with foo.work")
	if got := mustRead(t, filepath.Join(host, "foo.work")); got != "host\n" {
		t.Fatalf("host case-conflict file = %q, want untouched", got)
	}
	assertNotExist(t, filepath.Join(host, "bar.work"))
}

func TestLinkExistingHostCaseConflictWithParentVariantEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	requireCaseSensitiveFilesystem(t, tmp)
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "source\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "Nvim", "init.work.lua"), "host\n")

	result := runCubby(t, bin, host, "link")
	if result.code == 0 {
		t.Fatalf("link code = 0, want host case conflict failure")
	}
	assertContains(t, result.stdout, "CONFLICT nvim/init.work.lua")
	assertContains(t, result.stdout, "host path case conflict with Nvim")
	if got := mustRead(t, filepath.Join(host, "Nvim", "init.work.lua")); got != "host\n" {
		t.Fatalf("host parent-variant file = %q, want untouched", got)
	}
	assertNotExist(t, filepath.Join(host, "bar.work"))
}

func TestLinkExistingHostCaseConflictDryRunEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	requireCaseSensitiveFilesystem(t, tmp)
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "FOO.work"), "source\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	result := runCubby(t, bin, host, "link", "--dry-run")
	if result.code == 0 {
		t.Fatalf("link --dry-run code = 0, want host case conflict failure")
	}
	assertContains(t, result.stdout, "CONFLICT FOO.work")
	assertContains(t, result.stdout, "host path case conflict with foo.work")
	assertContains(t, result.stdout, "CREATE bar.work")
	if got := mustRead(t, filepath.Join(host, "foo.work")); got != "host\n" {
		t.Fatalf("host case-conflict file = %q, want untouched", got)
	}
	assertNotExist(t, filepath.Join(host, "bar.work"))
}

func TestLinkExistingHostCaseConflictIgnoreConflictsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	requireCaseSensitiveFilesystem(t, tmp)
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "FOO.work"), "source\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	result := runCubby(t, bin, host, "link", "--ignore-conflicts")
	if result.code != 0 {
		t.Fatalf("link --ignore-conflicts code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "SKIP FOO.work")
	assertContains(t, result.stdout, "host path case conflict with foo.work")
	if got := mustRead(t, filepath.Join(host, "foo.work")); got != "host\n" {
		t.Fatalf("host case-conflict file = %q, want untouched", got)
	}
	assertSymlinkExists(t, filepath.Join(host, "bar.work"))
}

func TestLinkExistingHostCaseSensitiveDryRunDoesNotReportPolicyConflictEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "FOO.work"), "source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	result := runCubby(t, bin, host, "link", "--dry-run", "--case-sensitive")
	if strings.Contains(result.stdout, "host path case conflict") {
		t.Fatalf("case-sensitive dry-run stdout = %q, want no host case-policy conflict", result.stdout)
	}
}

func TestLinkCaseSensitiveDryRunDoesNotReportCaseCollisionEndToEnd(t *testing.T) {
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
	mustWrite(t, filepath.Join(src1, "foo.work"), "one\n")
	mustWrite(t, filepath.Join(src2, "FOO.work"), "two\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	result := runCubby(t, bin, host, "link", "--dry-run", "--case-sensitive")
	if result.code != 0 {
		t.Fatalf("link --dry-run --case-sensitive code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertContains(t, result.stdout, "CREATE foo.work")
	assertContains(t, result.stdout, "CREATE FOO.work")
	if strings.Contains(result.stdout, "path case collision") || strings.Contains(result.stdout, "CONFLICT") || strings.Contains(result.stdout, "SKIP") {
		t.Fatalf("case-sensitive dry-run stdout = %q, want no case collision conflict/skip", result.stdout)
	}
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

func TestLinkInvalidSourceIgnorePatternFails(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\nignore = [\"[\"]\n")
	mustWrite(t, filepath.Join(src, "keep.work"), "source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"shared\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "link")
	assertFailureContains(t, result, "invalid ignore pattern")
	assertNotExist(t, filepath.Join(host, "keep.work"))
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

func TestMultiSourceLinkUnlinkWithMissingProfileDiagnosticsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	personalSrc := filepath.Join(tmp, "personal-src")
	workSrc := filepath.Join(tmp, "work-src")
	personalFile := filepath.Join(personalSrc, "personal.personal")
	workFile := filepath.Join(workSrc, "work.work")
	mustWrite(t, filepath.Join(personalSrc, "cubby.toml"), "profiles = [\"personal\"]\n")
	mustWrite(t, filepath.Join(workSrc, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, personalFile, "personal\n")
	mustWrite(t, workFile, "work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\", \"personal\"]\n\n[[source]]\nname = \"personal\"\npath = \""+personalSrc+"\"\n\n[[source]]\nname = \"work\"\npath = \""+workSrc+"\"\n")

	link := runCubby(t, bin, host, "link")
	if link.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", link.code, link.stdout, link.stderr)
	}
	assertContains(t, link.stderr, "source \"personal\" does not declare selected profile \"work\"; skipping")
	assertContains(t, link.stderr, "source \"work\" does not declare selected profile \"personal\"; skipping")
	assertSymlinkResolvesTo(t, filepath.Join(host, "personal.personal"), personalFile)
	assertSymlinkResolvesTo(t, filepath.Join(host, "work.work"), workFile)

	unlink := runCubby(t, bin, host, "unlink")
	if unlink.code != 0 {
		t.Fatalf("unlink code = %d, stdout = %s, stderr = %s", unlink.code, unlink.stdout, unlink.stderr)
	}
	assertContains(t, unlink.stderr, "source \"personal\" does not declare selected profile \"work\"; skipping")
	assertNotExist(t, filepath.Join(host, "personal.personal"))
	assertNotExist(t, filepath.Join(host, "work.work"))
}

func TestMultiSourceExplicitProfileAndDryRunDiagnosticsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	workSrc := filepath.Join(tmp, "work-src")
	clientSrc := filepath.Join(tmp, "client-src")
	mustWrite(t, filepath.Join(workSrc, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(clientSrc, "cubby.toml"), "profiles = [\"client\"]\n")
	mustWrite(t, filepath.Join(workSrc, "work.work"), "work\n")
	mustWrite(t, filepath.Join(clientSrc, "client.client"), "client\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"work\"\npath = \""+workSrc+"\"\n\n[[source]]\nname = \"client\"\npath = \""+clientSrc+"\"\n")

	dryRun := runCubby(t, bin, host, "link", "--dry-run", "--profile", "work", "--profile", "client")
	if dryRun.code != 0 {
		t.Fatalf("link --dry-run code = %d, stdout = %s, stderr = %s", dryRun.code, dryRun.stdout, dryRun.stderr)
	}
	assertContains(t, dryRun.stdout, "CREATE work.work")
	assertContains(t, dryRun.stdout, "CREATE client.client")
	assertContains(t, dryRun.stderr, "source \"work\" does not declare selected profile \"client\"; skipping")
	assertNotExist(t, filepath.Join(host, "work.work"))
	assertNotExist(t, filepath.Join(host, "client.client"))

	link := runCubby(t, bin, host, "link", "--profile", "work,client")
	if link.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", link.code, link.stdout, link.stderr)
	}
	assertSymlinkExists(t, filepath.Join(host, "work.work"))
	assertSymlinkExists(t, filepath.Join(host, "client.client"))
}
