package cmd

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

func executeForTest(args ...string) (stdout string, stderr string, err error) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(&out, &errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}
