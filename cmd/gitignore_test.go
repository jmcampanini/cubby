package cmd

import (
	"strings"
	"testing"
)

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
