package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// pflagSingularHelpSuffix is the fixed text go-config-loader's pflagloader
// appends to the usage of every pflag_singular flag. The repository cannot
// shorten it, so the help test measures flag lines without it.
const pflagSingularHelpSuffix = "Adds a single value to the array; empty values are not allowed."

func TestRootHelpShowsV01CommandSurface(t *testing.T) {
	out, errOut, err := executeForTest("--help")
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, errOut)
	}

	for _, want := range []string{
		"link", "unlink", "prune", "status", "doctor",
		"profile", "source", "gitignore", "lazygit", "config",
		"Help topics:", "exit-codes", "workflow", "--color",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("root help missing %q:\n%s", want, out)
		}
	}
}

// TestEveryApplicationCommandRendersCompleteHelp walks a fresh tree and checks
// the help baseline on the help each command renders for --help rather than
// on the raw fields: a short help, a long description, every line within 80
// columns (flag usage included, apart from the library suffix the repository
// cannot shorten), and, for a help topic, a direct invocation that prints
// exactly what 'cubby help <topic>' prints.
func TestEveryApplicationCommandRendersCompleteHelp(t *testing.T) {
	root := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{})

	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Name() == "help" || command.Name() == "completion" {
			return
		}
		path := command.CommandPath()
		if strings.TrimSpace(command.Short) == "" {
			t.Errorf("%q has no short help", path)
		}
		if strings.TrimSpace(command.Long) == "" {
			t.Errorf("%q has no long description", path)
		}

		invocation := strings.Fields(path)[1:]
		help, errOut, err := executeForTest(append(invocation, "--help")...)
		if err != nil {
			t.Fatalf("%q --help error = %v, stderr = %s", path, err, errOut)
		}
		for i, line := range strings.Split(help, "\n") {
			measured := strings.TrimSuffix(line, " "+pflagSingularHelpSuffix)
			if width := utf8.RuneCountInString(measured); width > 80 {
				t.Errorf("%q help line %d is %d columns, want at most 80: %q", path, i+1, width, line)
			}
		}

		if command.GroupID == helpTopicGroupID {
			direct, errOut, err := executeForTest(invocation...)
			if err != nil {
				t.Fatalf("%q error = %v, stderr = %s", path, err, errOut)
			}
			viaHelp, errOut, err := executeForTest(append([]string{"help"}, invocation...)...)
			if err != nil {
				t.Fatalf("help %q error = %v, stderr = %s", path, err, errOut)
			}
			if direct != viaHelp {
				t.Errorf("%q invoked directly prints text other than its help:\n%s\n---\n%s", path, direct, viaHelp)
			}
		}

		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(root)
}

func TestBareRootAndCompletionPrintTheirHelp(t *testing.T) {
	for _, args := range [][]string{{}, {"completion"}} {
		t.Run(strings.Join(append([]string{"cubby"}, args...), " "), func(t *testing.T) {
			var help, stdout, stderr bytes.Buffer
			if code := Execute(append(args, "--help"), &help, &bytes.Buffer{}); code != 0 {
				t.Fatalf("Execute(%v --help) = %d, want 0", args, code)
			}

			code := Execute(args, &stdout, &stderr)

			if code != 0 || stderr.Len() != 0 {
				t.Errorf("Execute(%v) = %d, stderr %q; want 0 and no diagnostic", args, code, stderr.String())
			}
			if stdout.String() != help.String() {
				t.Errorf("Execute(%v) stdout = %q, want its --help text %q", args, stdout.String(), help.String())
			}
		})
	}
}

func TestExecuteExitsTwoOnUsageErrorsWithOneLine(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown command", args: []string{"bogus"}, want: `cubby: unknown command "bogus" for "cubby"` + "\n"},
		{name: "root operand", args: []string{"--", "bogus"}, want: `cubby: unknown command "bogus" for "cubby"` + "\n"},
		{name: "unknown help topic", args: []string{"help", "bogus"}, want: `cubby: unknown command "bogus" for "cubby"` + "\n"},
		{name: "unknown nested help topic", args: []string{"help", "profile", "bogus"}, want: `cubby: unknown command "bogus" for "cubby profile"` + "\n"},
		{name: "extra help operand", args: []string{"help", "status", "bogus"}, want: `cubby: unknown command "bogus" for "cubby status"` + "\n"},
		{name: "unknown completion shell", args: []string{"completion", "bogus"}, want: `cubby: unknown command "bogus" for "cubby completion"` + "\n"},
		{name: "unknown command before --help", args: []string{"bogus", "--help"}, want: `cubby: unknown command "bogus" for "cubby"` + "\n"},
		{name: "unknown command before --version", args: []string{"bogus", "--version"}, want: `cubby: unknown command "bogus" for "cubby"` + "\n"},
		{name: "unknown flag", args: []string{"status", "--bogus"}, want: "cubby: unknown flag: --bogus\n"},
		{name: "newline in the message", args: []string{"status", "--bogus\nflag"}, want: "cubby: unknown flag: --bogus flag\n"},
		{name: "color outside its set", args: []string{"--color=bold", "source", "list"}, want: `cubby: invalid argument "bold" for "--color" flag: unknown color mode "bold"; use auto, always, or never` + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			code := Execute(tt.args, &out, &errOut)

			if code != 2 {
				t.Errorf("Execute(%v) = %d, want 2", tt.args, code)
			}
			if got := errOut.String(); got != tt.want {
				t.Errorf("Execute(%v) stderr = %q, want %q", tt.args, got, tt.want)
			}
			if out.Len() != 0 {
				t.Errorf("Execute(%v) stdout = %q, want empty", tt.args, out.String())
			}
		})
	}
}

func TestExitCodesTopicListsEveryStatus(t *testing.T) {
	out, errOut, err := executeForTest("exit-codes")
	if err != nil {
		t.Fatalf("exit-codes error = %v, stderr = %s", err, errOut)
	}
	for _, want := range []string{"\n  0  ", "\n  1  ", "\n  2  ", "\n  N  ", "'cubby: <message>'"} {
		if !strings.Contains(out, want) {
			t.Errorf("exit-codes help missing %q:\n%s", want, out)
		}
	}
}

func executeForTest(args ...string) (stdout string, stderr string, err error) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := newRootCommand(&out, &errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}
