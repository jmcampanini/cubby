package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStatusCommandReportsManagedLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustSymlinkForCmdTest(t, filepath.Join(host, "nvim", "init.work.lua"), filepath.Join(src, "nvim", "init.work.lua"))
	mustChdir(t, host)

	out, _, err := executeForTest("status")
	if err != nil {
		t.Fatalf("status error = %v", err)
	}
	for _, want := range []string{"LINK nvim/init.work.lua", "source=src", "profile=work", "target=nvim/init.work.lua"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
}

func TestPruneRemovesDanglingManagedSymlinksOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	managed := filepath.Join(host, "stale.work")
	unmanaged := filepath.Join(host, "outside.work")
	valid := filepath.Join(host, "valid.work")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "valid.work"), "valid\n")
	mustWrite(t, filepath.Join(other, "missing.work"), "temp\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustSymlinkForCmdTest(t, managed, filepath.Join(src, "stale.work"))
	mustSymlinkForCmdTest(t, unmanaged, filepath.Join(other, "gone.work"))
	mustSymlinkForCmdTest(t, valid, filepath.Join(src, "valid.work"))
	mustChdir(t, host)

	out, _, err := executeForTest("prune")
	if err != nil {
		t.Fatalf("prune error = %v", err)
	}
	if out != "stale.work\n" {
		t.Fatalf("prune output = %q, want stale path", out)
	}
	if _, err := os.Lstat(managed); !os.IsNotExist(err) {
		t.Fatalf("managed stale Lstat error = %v, want not exist", err)
	}
	for _, path := range []string{unmanaged, valid} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s was not preserved as symlink: info=%v err=%v", path, info, err)
		}
	}
}

func TestPruneRemovesDanglingLinksForMissingSourceRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	managed := filepath.Join(host, "stale.work")
	unmanaged := filepath.Join(host, "outside.work")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "stale.work"), "stale\n")
	mustWrite(t, filepath.Join(other, "placeholder"), "other\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustSymlinkForCmdTest(t, managed, filepath.Join(src, "stale.work"))
	mustSymlinkForCmdTest(t, unmanaged, filepath.Join(other, "gone.work"))
	if err := os.RemoveAll(src); err != nil {
		t.Fatalf("RemoveAll(%q) error = %v", src, err)
	}
	mustChdir(t, host)

	out, _, err := executeForTest("prune")
	if err != nil {
		t.Fatalf("prune error = %v", err)
	}
	if out != "stale.work\n" {
		t.Fatalf("prune output = %q, want stale path", out)
	}
	if _, err := os.Lstat(managed); !os.IsNotExist(err) {
		t.Fatalf("managed stale Lstat error = %v, want not exist", err)
	}
	if info, err := os.Lstat(unmanaged); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("unmanaged symlink was not preserved: info=%v err=%v", info, err)
	}
}

func TestDoctorReportsHealthyAndUnhealthySetups(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "ok.work"), "ok\n")

	t.Run("healthy", func(t *testing.T) {
		host := filepath.Join(root, "healthy")
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
		mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
		mustSymlinkForCmdTest(t, filepath.Join(host, "ok.work"), filepath.Join(src, "ok.work"))
		mustChdir(t, host)

		out, _, err := executeForTest("doctor")
		if err != nil {
			t.Fatalf("doctor healthy error = %v output = %s", err, out)
		}
		if out != "" {
			t.Fatalf("doctor healthy output = %q, want empty", out)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		host := filepath.Join(root, "unhealthy")
		mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\", \"ghost\"]\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n\n[[source]]\nname = \"missing\"\npath = \"../missing\"\n")
		mustWrite(t, filepath.Join(host, "ok.work"), "conflict\n")
		mustSymlinkForCmdTest(t, filepath.Join(host, "stale.work"), filepath.Join(src, "stale.work"))
		mustChdir(t, host)

		out, _, err := executeForTest("doctor")
		if err == nil {
			t.Fatalf("doctor unhealthy error = nil, output = %s", out)
		}
		for _, want := range []string{"MISSING_SOURCE", "MISSING_GITIGNORE", "MISSING_PROFILE ghost", "DANGLING stale.work", "CONFLICT ok.work"} {
			if !strings.Contains(out, want) {
				t.Fatalf("doctor output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestDoctorReportsUnresolvedAndNonRegularManagedLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	if err := os.Symlink("loop.work", filepath.Join(src, "loop.work")); err != nil {
		t.Fatalf("Symlink(loop.work) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "dir.work"), 0o755); err != nil {
		t.Fatalf("MkdirAll(dir.work) error = %v", err)
	}
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustSymlinkForCmdTest(t, filepath.Join(host, "loop.work"), filepath.Join(src, "loop.work"))
	mustSymlinkForCmdTest(t, filepath.Join(host, "dir.work"), filepath.Join(src, "dir.work"))
	mustChdir(t, host)

	out, _, err := executeForTest("doctor")
	if err == nil {
		t.Fatalf("doctor error = nil, output = %s", out)
	}
	for _, want := range []string{"DRIFT loop.work", "unresolved target", "DRIFT dir.work", "non-regular target"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorReportsInvalidSourceIgnorePattern(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\nignore = [\"[\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustChdir(t, host)

	out, _, err := executeForTest("doctor")
	if err == nil {
		t.Fatalf("doctor error = nil, output = %s", out)
	}
	for _, want := range []string{"MISSING_SOURCE src", "invalid ignore pattern"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorReportsConflictsForSymlinkedSourceRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	realSrc := filepath.Join(root, "real-src")
	hostSrc := filepath.Join(host, "src")
	mustWrite(t, filepath.Join(realSrc, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(realSrc, "conflict.work"), "source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \"./src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host\n")
	mustSymlinkForCmdTest(t, hostSrc, realSrc)
	mustChdir(t, host)

	out, _, err := executeForTest("doctor")
	if err == nil {
		t.Fatalf("doctor error = nil, output = %s", out)
	}
	if !strings.Contains(out, "CONFLICT conflict.work") {
		t.Fatalf("doctor output = %q, want symlinked source conflict", out)
	}
}

func TestDoctorReportsConflictsWhenHostIgnoresConflicts(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nignore_conflicts = true\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host\n")
	mustChdir(t, host)

	out, _, err := executeForTest("doctor")
	if err == nil {
		t.Fatalf("doctor error = nil, output = %s", out)
	}
	if !strings.Contains(out, "CONFLICT conflict.work") {
		t.Fatalf("doctor output = %q, want conflict despite ignore_conflicts", out)
	}
}

func TestDoctorUsesEffectiveProfileSelectionFromEnvAndFlags(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustChdir(t, host)

	t.Run("env", func(t *testing.T) {
		t.Setenv("CUBBY_PROFILE", "ghost")
		out, _, err := executeForTest("doctor")
		if err == nil {
			t.Fatalf("doctor env error = nil, output = %s", out)
		}
		if !strings.Contains(out, "MISSING_PROFILE ghost") {
			t.Fatalf("doctor env output = %q, want missing env profile", out)
		}
	})

	t.Run("flag", func(t *testing.T) {
		t.Setenv("CUBBY_PROFILE", "work")
		out, _, err := executeForTest("doctor", "--profile", "ghost")
		if err == nil {
			t.Fatalf("doctor flag error = nil, output = %s", out)
		}
		if !strings.Contains(out, "MISSING_PROFILE ghost") {
			t.Fatalf("doctor flag output = %q, want missing flag profile", out)
		}
	})
}

func mustSymlinkForCmdTest(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(linkPath), err)
	}
	target, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", target, linkPath, err)
	}
}
