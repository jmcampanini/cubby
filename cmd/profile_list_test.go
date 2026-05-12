package cmd

import (
	"path/filepath"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestProfileListPrintsSourceDeclaredProfilesSortedDeduped(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src1 := filepath.Join(root, "src1")
	src2 := filepath.Join(root, "src2")
	mustWrite(t, filepath.Join(src1, config.SourceConfigFileName), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, config.SourceConfigFileName), "profiles = [\"client\", \"work\"]\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), "profiles = [\"host-only\", \"work\"]\n\n[[source]]\nname = \"one\"\npath = \"../src1\"\n\n[[source]]\nname = \"two\"\npath = \"../src2\"\n")
	mustChdir(t, host)

	out, errOut, err := executeForTest("profile", "list")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	want := "client\npersonal\nwork\n"
	if out != want {
		t.Fatalf("profile list output = %q, want %q", out, want)
	}
}
