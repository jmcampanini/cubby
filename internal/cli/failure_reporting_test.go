package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportFailuresKeepPayloadAndPrintCountedDiagnostic(t *testing.T) {
	// Each case sets up count failures: count of the three required patterns
	// missing from .gitignore for doctor and gitignore check, and count host
	// files that block a link.
	tests := []struct {
		name  string
		args  []string
		count int
		want  string
	}{
		{name: "doctor", args: []string{"doctor"}, count: 1, want: "doctor found 1 issue"},
		{name: "doctor", args: []string{"doctor"}, count: 2, want: "doctor found 2 issues"},
		{name: "gitignore", args: []string{"gitignore", "check"}, count: 1, want: ".gitignore is missing 1 pattern"},
		{name: "gitignore", args: []string{"gitignore", "check"}, count: 2, want: ".gitignore is missing 2 patterns"},
		{name: "link", args: []string{"link"}, count: 1, want: "1 fatal link conflict"},
		{name: "link", args: []string{"link"}, count: 2, want: "2 fatal link conflicts"},
		{name: "link dry run", args: []string{"link", "--dry-run"}, count: 1, want: "1 fatal link conflict"},
		{name: "link dry run", args: []string{"link", "--dry-run"}, count: 2, want: "2 fatal link conflicts"},
	}
	for _, tt := range tests {
		for _, jsonOutput := range []bool{false, true} {
			count, message := tt.count, tt.want
			t.Run(fmt.Sprintf("%s/json=%t/count=%d", tt.name, jsonOutput, count), func(t *testing.T) {
				root := t.TempDir()
				host := filepath.Join(root, "host")
				src := filepath.Join(root, "src")
				mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
				mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
				patterns := []string{"/.cubby.toml", "*.work.*", "*.work"}
				mustWrite(t, filepath.Join(host, ".gitignore"), strings.Join(patterns[count:], "\n")+"\n")
				if tt.args[0] == "link" {
					for i := range count {
						path := fmt.Sprintf("file%d.work", i)
						mustWrite(t, filepath.Join(src, path), "source\n")
						mustWrite(t, filepath.Join(host, path), "host\n")
					}
				}
				mustChdir(t, host)
				args := append([]string(nil), tt.args...)
				if jsonOutput {
					args = append(args, "--json")
				}
				var stdout, stderr bytes.Buffer

				code := Execute(args, &stdout, &stderr)

				if code != 1 {
					t.Errorf("Execute(%v) = %d, want 1", args, code)
				}
				if want := "cubby: " + message + "\n"; stderr.String() != want {
					t.Errorf("stderr = %q, want %q", stderr.String(), want)
				}
				if strings.Contains(stdout.String(), message) {
					t.Errorf("stdout contains failure diagnostic: %q", stdout.String())
				}
				if jsonOutput {
					var report map[string]json.RawMessage
					if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
						t.Fatalf("decode stdout %q: %v", stdout.String(), err)
					}
					key := map[string]string{"doctor": "issues", "gitignore": "missing", "link": "actions"}[tt.args[0]]
					var items []json.RawMessage
					if err := json.Unmarshal(report[key], &items); err != nil {
						t.Fatalf("decode %s: %v", key, err)
					}
					if len(items) != count || len(report) != 2 {
						t.Errorf("report = %s, want %d %s and two envelope fields", stdout.String(), count, key)
					}
				} else if got := strings.Count(stdout.String(), "\n"); got != count {
					t.Errorf("stdout = %q, want %d report lines", stdout.String(), count)
				}
			})
		}
	}
}

func TestSuccessfulReportsHaveNoFailureDiagnostic(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	mustWrite(t, filepath.Join(root, "src", "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n[[source]]\nname = \"src\"\npath = \"../src\"\n")
	mustWrite(t, filepath.Join(host, ".gitignore"), "/.cubby.toml\n*.work.*\n*.work\n")
	mustChdir(t, host)

	for _, args := range [][]string{{"doctor"}, {"gitignore", "check"}, {"link"}, {"link", "--dry-run"}} {
		for _, jsonOutput := range []bool{false, true} {
			invocation := append([]string(nil), args...)
			if jsonOutput {
				invocation = append(invocation, "--json")
			}
			t.Run(strings.Join(invocation, " "), func(t *testing.T) {
				var stdout, stderr bytes.Buffer

				code := Execute(invocation, &stdout, &stderr)

				if code != 0 || stderr.Len() != 0 {
					t.Errorf("Execute(%v) = %d, stderr %q; want 0 and no diagnostic", invocation, code, stderr.String())
				}
			})
		}
	}
}
