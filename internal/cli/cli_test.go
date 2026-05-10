package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpShowsV01CommandSurface(t *testing.T) {
	out, errOut, err := executeForTest("--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}

	for _, want := range []string{
		"link", "unlink", "prune", "status", "doctor",
		"profile", "source", "gitignore", "lazygit",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("root help missing %q:\n%s", want, out)
		}
	}
}

func TestGitignoreHelpShowsCheckAndSync(t *testing.T) {
	out, errOut, err := executeForTest("gitignore", "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"check", "sync"} {
		if !strings.Contains(out, want) {
			t.Fatalf("gitignore help missing %q:\n%s", want, out)
		}
	}
}

func TestGitignoreSubcommandHelpIsUseful(t *testing.T) {
	for _, args := range [][]string{{"gitignore", "check", "--help"}, {"gitignore", "sync", "--help"}} {
		out, errOut, err := executeForTest(args...)
		if err != nil {
			t.Fatalf("Execute(%v) error = %v, stderr = %s", args, err, errOut)
		}
		if !strings.Contains(out, ".gitignore") || !strings.Contains(out, "profiles") {
			t.Fatalf("help for %v not useful:\n%s", args, out)
		}
	}
}

func TestNotImplementedCommandExitsWithClearError(t *testing.T) {
	_, _, err := executeForTest("status")
	if err == nil {
		t.Fatalf("status error = nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("status error = %q, want not implemented", err)
	}
}

func executeForTest(args ...string) (stdout string, stderr string, err error) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(&out, &errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}
