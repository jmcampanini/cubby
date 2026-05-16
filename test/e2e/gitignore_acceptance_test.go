package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestGitignoreCheckSyncMultiSourceUnionEndToEnd(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"client\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	checkBefore := runCubby(t, bin, host, "gitignore", "check")
	if checkBefore.code == 0 {
		t.Fatalf("gitignore check code = 0, want missing patterns")
	}
	for _, want := range []string{"*.client.*", "*.client", "*.personal.*", "*.personal", "*.work.*", "*.work"} {
		assertContains(t, checkBefore.stdout, want)
	}
	sync := runCubby(t, bin, host, "gitignore", "sync")
	if sync.code != 0 {
		t.Fatalf("gitignore sync code = %d, stdout = %s, stderr = %s", sync.code, sync.stdout, sync.stderr)
	}
	content := mustRead(t, filepath.Join(host, ".gitignore"))
	for _, want := range []string{"*.client.*\n", "*.client\n", "*.personal.*\n", "*.personal\n", "*.work.*\n", "*.work\n"} {
		assertContains(t, content, want)
	}
	checkAfter := runCubby(t, bin, host, "gitignore", "check")
	if checkAfter.code != 0 {
		t.Fatalf("gitignore check after sync code = %d, stdout = %s, stderr = %s", checkAfter.code, checkAfter.stdout, checkAfter.stderr)
	}
	syncAgain := runCubby(t, bin, host, "gitignore", "sync")
	if syncAgain.code != 0 || syncAgain.stdout != "" {
		t.Fatalf("second gitignore sync code = %d stdout = %q stderr = %s", syncAgain.code, syncAgain.stdout, syncAgain.stderr)
	}
}
