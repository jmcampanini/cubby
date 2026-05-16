package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestSourceListJSONPrintsRegisteredSourcesInOrder(t *testing.T) {
	host, src1, src2 := writeSourceListProject(t)
	mustChdir(t, host)

	out, errOut, err := executeForTest("source", "list", "--json")
	if err != nil {
		t.Fatalf("source list --json error = %v, stderr = %s", err, errOut)
	}
	want := `{"sources":[{"name":"one","path":"` + filepath.ToSlash(resolvedPath(t, src1)) + `","profiles":["work","personal"]},{"name":"two","path":"` + filepath.ToSlash(resolvedPath(t, src2)) + `","profiles":["client"]}]}` + "\n"
	if out != want {
		t.Fatalf("source list --json output = %q, want %q", out, want)
	}
	if errOut != "" {
		t.Fatalf("source list --json stderr = %q, want empty", errOut)
	}
}

func TestSourceListDefaultTableIncludesInventory(t *testing.T) {
	host, src1, src2 := writeSourceListProject(t)
	mustChdir(t, host)

	out, errOut, err := executeForTest("source", "list")
	if err != nil {
		t.Fatalf("source list error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"NAME", "PATH", "PROFILES", "one", resolvedPath(t, src1), "work,personal", "two", resolvedPath(t, src2), "client"} {
		if !strings.Contains(out, want) {
			t.Fatalf("source list output missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "one") > strings.Index(out, "two") {
		t.Fatalf("source list output order = %q, want host registration order", out)
	}
}

func TestSourceListFailsStrictLoadingForInvalidSource(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, config.SourceConfigFileName), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), "[[source]]\nname = \"bad.name\"\npath = \"../src\"\n")
	mustChdir(t, host)

	_, _, err := executeForTest("source", "list")
	if err == nil {
		t.Fatalf("source list error = nil, want invalid source failure")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("source list error = %q, want invalid", err)
	}
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return resolved
}

func writeSourceListProject(t *testing.T) (host, src1, src2 string) {
	t.Helper()
	root := t.TempDir()
	host = filepath.Join(root, "host")
	src1 = filepath.Join(root, "src1")
	src2 = filepath.Join(root, "src2")
	mustWrite(t, filepath.Join(src1, config.SourceConfigFileName), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, config.SourceConfigFileName), "profiles = [\"client\"]\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), "profiles = [\"host-only\"]\n\n[[source]]\nname = \"one\"\npath = \"../src1\"\n\n[[source]]\nname = \"two\"\npath = \"../src2\"\n")
	return host, src1, src2
}
