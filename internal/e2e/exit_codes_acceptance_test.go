package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExitCodesAcceptanceUsageErrorsExitTwoWithOneLine(t *testing.T) {
	bin := buildCubby(t)
	host := t.TempDir()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root command", args: []string{"bogus"}, want: `unknown command "bogus"`},
		{name: "group command", args: []string{"source", "bogus"}, want: `unknown command "bogus"`},
		{name: "operand", args: []string{"link", "bogus"}, want: `unknown command "bogus"`},
		{name: "topic operand", args: []string{"workflow", "bogus"}, want: `unknown command "bogus"`},
		{name: "flag", args: []string{"status", "--bogus"}, want: "unknown flag: --bogus"},
		{name: "flag value", args: []string{"link", "--dry-run=bogus"}, want: `invalid argument "bogus"`},
		{name: "color value", args: []string{"--color=bold", "source", "list"}, want: `unknown color mode "bold"`},
		{name: "missing value", args: []string{"lazygit", "--source"}, want: "flag needs an argument: --source"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCubby(t, bin, host, tt.args...)

			if result.code != 2 || result.stdout != "" {
				t.Fatalf("cubby %v = code %d, stdout %q, stderr %q; want code 2 and no stdout", tt.args, result.code, result.stdout, result.stderr)
			}
			if !strings.HasPrefix(result.stderr, "cubby: ") || !strings.Contains(result.stderr, tt.want) ||
				strings.Count(result.stderr, "\n") != 1 || strings.Contains(result.stderr, "Usage:") || strings.Contains(result.stderr, "rror:") {
				t.Errorf("stderr = %q, want one 'cubby: <message>' line containing %q", result.stderr, tt.want)
			}
		})
	}
}

func TestExitCodesAcceptanceCommandFailureExitsOneWithOneLine(t *testing.T) {
	bin := buildCubby(t)

	result := runCubby(t, bin, t.TempDir(), "status")

	if result.code != 1 || result.stdout != "" {
		t.Fatalf("status = code %d, stdout %q, stderr %q; want code 1 and no stdout", result.code, result.stdout, result.stderr)
	}
	if !strings.HasPrefix(result.stderr, "cubby: ") || !strings.Contains(result.stderr, ".cubby.toml") ||
		strings.Count(result.stderr, "\n") != 1 || strings.Contains(result.stderr, "Usage:") || strings.Contains(result.stderr, "rror:") {
		t.Errorf("stderr = %q, want one 'cubby: <message>' line naming .cubby.toml", result.stderr)
	}
}

func TestExitCodesAcceptanceReportFailuresEndWithOneLine(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(src, "create.work"), "source create\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")
	tests := []struct {
		args       []string
		wantStderr string
	}{
		{args: []string{"doctor"}, wantStderr: "cubby: doctor found 4 issues\n"},
		{args: []string{"doctor", "--json"}, wantStderr: "cubby: doctor found 4 issues\n"},
		{args: []string{"gitignore", "check"}, wantStderr: "cubby: .gitignore is missing 3 patterns\n"},
		{args: []string{"gitignore", "check", "--json"}, wantStderr: "cubby: .gitignore is missing 3 patterns\n"},
		{args: []string{"link"}, wantStderr: "cubby: 1 fatal link conflict\n"},
		{args: []string{"link", "--dry-run"}, wantStderr: "cubby: 1 fatal link conflict\n"},
		{args: []string{"link", "--json"}, wantStderr: "cubby: 1 fatal link conflict\n"},
		{args: []string{"link", "--json", "--dry-run"}, wantStderr: "cubby: 1 fatal link conflict\n"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result := runCubby(t, bin, host, tt.args...)

			if result.code != 1 || result.stdout == "" {
				t.Fatalf("cubby %v = code %d, stdout %q; want code 1 and a report on stdout", tt.args, result.code, result.stdout)
			}
			if strings.Contains(result.stdout, "cubby:") {
				t.Errorf("cubby %v stdout = %q, want no diagnostic", tt.args, result.stdout)
			}
			if result.stderr != tt.wantStderr {
				t.Errorf("cubby %v stderr = %q, want %q", tt.args, result.stderr, tt.wantStderr)
			}
		})
	}
	assertNotExist(t, filepath.Join(host, "create.work"))
}
