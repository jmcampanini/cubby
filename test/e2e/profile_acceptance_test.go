package e2e_test

import (
	"path/filepath"
	"runtime"
	"testing"
)

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

func TestMultiSourceEmptyHostProfilesRequireExplicitSelectionEndToEnd(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"personal\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	link := runCubby(t, bin, host, "link")
	assertFailureContains(t, link, "no profiles selected")
	unlink := runCubby(t, bin, host, "unlink")
	assertFailureContains(t, unlink, "no profiles selected")
}

func TestProfileEffectivePrintsHostProfilesEndToEnd(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "profile", "effective")
	if result.code != 0 {
		t.Fatalf("profile effective code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "work\n" {
		t.Fatalf("profile effective stdout = %q, want %q", result.stdout, "work\n")
	}
}

func TestProfileEffectiveAppendsEnvProfiles(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nenv_profiles = \"CUBBY_EXTRA\"\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_EXTRA": "personal,work"}, "profile", "effective")
	if result.code != 0 {
		t.Fatalf("profile effective code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "work\npersonal\n" {
		t.Fatalf("profile effective stdout = %q, want %q (dedupe first-seen)", result.stdout, "work\npersonal\n")
	}
}

func TestProfileEffectiveFlagReplacesThenEnvProfilesAppends(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\", \"personal\", \"client\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nenv_profiles = \"CUBBY_EXTRA\"\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_EXTRA": "personal"}, "profile", "effective", "--profile", "client")
	if result.code != 0 {
		t.Fatalf("profile effective code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "client\npersonal\n" {
		t.Fatalf("profile effective stdout = %q, want %q", result.stdout, "client\npersonal\n")
	}
}

func TestProfileEffectiveEmptyWarnsAndExitsZero(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubby(t, bin, host, "profile", "effective")
	if result.code != 0 {
		t.Fatalf("profile effective code = %d, want 0; stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("profile effective stdout = %q, want empty", result.stdout)
	}
	assertContains(t, result.stderr, "no profiles selected")
}

func TestLinkAppliesEnvProfiles(t *testing.T) {
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
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nenv_profiles = \"CUBBY_EXTRA\"\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_EXTRA": "personal"}, "link")
	if result.code != 0 {
		t.Fatalf("link code = %d, stdout = %s, stderr = %s", result.code, result.stdout, result.stderr)
	}
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.work.lua"))
	assertSymlinkExists(t, filepath.Join(host, "nvim", "init.personal.lua"))
}

func TestLinkRejectsUndeclaredEnvProfilesContribution(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "nvim", "init.work.lua"), "-- work\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\nenv_profiles = \"CUBBY_EXTRA\"\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	result := runCubbyEnv(t, bin, host, map[string]string{"CUBBY_EXTRA": "ghost"}, "link")
	assertFailureContains(t, result, "ghost")
	assertFailureContains(t, result, "not declared")
	assertNotExist(t, filepath.Join(host, "nvim", "init.work.lua"))
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
