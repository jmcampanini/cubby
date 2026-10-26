package e2e_test

import (
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
