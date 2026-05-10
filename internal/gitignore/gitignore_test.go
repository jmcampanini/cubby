package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredPatterns(t *testing.T) {
	got := RequiredPatterns([]string{"client", "work"})
	want := []string{"*.client.*", "*.client", "*.work.*", "*.work"}
	assertStringSlicesEqual(t, got, want)
}

func TestMissingPatternsMatchesExactTrimmedNonCommentLines(t *testing.T) {
	content := []byte(strings.Join([]string{
		"# *.work.*",
		"  *.work  ",
		"*.client.* # inline text makes this a different exact line",
		"*.client.*",
	}, "\n"))
	required := []string{"*.work.*", "*.work", "*.client.*", "*.client"}

	got := MissingPatterns(content, required)
	want := []string{"*.work.*", "*.client"}
	assertStringSlicesEqual(t, got, want)
}

func TestAppendMissingCreatesAndDoesNotDuplicateWhenRechecked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	required := []string{"*.work.*", "*.work"}

	missing, err := MissingPatternsFile(path, required)
	if err != nil {
		t.Fatalf("MissingPatternsFile() error = %v", err)
	}
	assertStringSlicesEqual(t, missing, required)

	if err := AppendMissing(path, missing); err != nil {
		t.Fatalf("AppendMissing() error = %v", err)
	}
	missing, err = MissingPatternsFile(path, required)
	if err != nil {
		t.Fatalf("MissingPatternsFile() after append error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing after append = %v, want none", missing)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := strings.Count(string(content), "*.work.*"); got != 1 {
		t.Fatalf("*.work.* count = %d, want 1", got)
	}
}

func TestAppendMissingPreservesReadableNewlines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := AppendMissing(path, []string{"*.work.*", "*.work"}); err != nil {
		t.Fatalf("AppendMissing() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "existing\n*.work.*\n*.work\n"
	if string(content) != want {
		t.Fatalf("content = %q, want %q", content, want)
	}
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
