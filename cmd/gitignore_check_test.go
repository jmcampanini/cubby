package cmd

import (
	"strings"
	"testing"
)

func TestGitignoreCheckHelpIsUseful(t *testing.T) {
	out, errOut, err := executeForTest("gitignore", "check", "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if !strings.Contains(out, ".gitignore") || !strings.Contains(out, "profiles") {
		t.Fatalf("gitignore check help not useful:\n%s", out)
	}
}
