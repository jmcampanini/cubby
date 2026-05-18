package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestGitignoreCheckUsesAllSourceDeclaredProfiles(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src1 := filepath.Join(root, "src1")
	src2 := filepath.Join(root, "src2")
	mustWrite(t, filepath.Join(src1, config.SourceConfigFileName), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, config.SourceConfigFileName), "profiles = [\"client\"]\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), "profiles = [\"work\"]\n\n[[source]]\nname = \"one\"\npath = \"../src1\"\n\n[[source]]\nname = \"two\"\npath = \"../src2\"\n")
	mustChdir(t, host)

	out, _, err := executeForTest("gitignore", "check")
	if err == nil {
		t.Fatalf("gitignore check error = nil, want missing patterns")
	}
	for _, want := range []string{"/.cubby.toml", "*.client.*", "*.client", "*.personal.*", "*.personal", "*.work.*", "*.work"} {
		if !strings.Contains(out, want) {
			t.Fatalf("gitignore check output missing %q:\n%s", want, out)
		}
	}
}

func TestGitignoreCheckRequiresHostConfigEvenWhenProfilePatternsPresent(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, config.SourceConfigFileName), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), "[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "*.work.*\n*.work\n")
	mustChdir(t, host)

	out, _, err := executeForTest("gitignore", "check")
	if err == nil {
		t.Fatalf("gitignore check error = nil, want missing host config pattern")
	}
	if !strings.Contains(out, "/.cubby.toml") {
		t.Fatalf("gitignore check output missing %q:\n%s", "/.cubby.toml", out)
	}
	for _, unwanted := range []string{"*.work.*", "*.work"} {
		if strings.Contains(out, unwanted+"\n") {
			t.Fatalf("gitignore check output unexpectedly lists %q:\n%s", unwanted, out)
		}
	}
}

func TestGitignoreCheckHelpIsUseful(t *testing.T) {
	out, errOut, err := executeForTest("gitignore", "check", "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if !strings.Contains(out, ".gitignore") || !strings.Contains(out, "profiles") {
		t.Fatalf("gitignore check help not useful:\n%s", out)
	}
}
