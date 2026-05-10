package cmd

import (
	"strings"
	"testing"
)

func TestGitignoreSyncHelpIsUseful(t *testing.T) {
	out, errOut, err := executeForTest("gitignore", "sync", "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}
	if !strings.Contains(out, ".gitignore") || !strings.Contains(out, "profiles") {
		t.Fatalf("gitignore sync help not useful:\n%s", out)
	}
}
