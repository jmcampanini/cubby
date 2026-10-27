package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

type fakeExternalRunner struct {
	calls []externalCommand
	err   error
}

func (r *fakeExternalRunner) run(command externalCommand) error {
	recorded := command
	if command.Args != nil {
		recorded.Args = append([]string(nil), command.Args...)
	}
	r.calls = append(r.calls, recorded)
	return r.err
}

func TestLazygitSingleRegisteredSourceSelectedImplicitly(t *testing.T) {
	host, sources := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	out, errOut, err := executeForTest("lazygit")
	if err != nil {
		t.Fatalf("lazygit error = %v, stderr = %s", err, errOut)
	}
	if out != "" || errOut != "" {
		t.Fatalf("lazygit output = %q stderr = %q, want empty", out, errOut)
	}
	assertLazygitCall(t, runner, sources["one"])
}

func TestLazygitMultipleSourcesWithoutSourceFailsAmbiguous(t *testing.T) {
	host, _ := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	_, _, err := executeForTest("lazygit")
	if err == nil {
		t.Fatalf("lazygit error = nil, want ambiguity")
	}
	for _, want := range []string{"multiple sources", "--source", "one", "two"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("lazygit error = %q, want %q", err, want)
		}
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestLazygitMultipleSourcesWithSourceRunsSelected(t *testing.T) {
	host, sources := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	_, errOut, err := executeForTest("lazygit", "--source", "two")
	if err != nil {
		t.Fatalf("lazygit --source two error = %v, stderr = %s", err, errOut)
	}
	assertLazygitCall(t, runner, sources["two"])
}

func TestLazygitSourceFlagTrimsWhitespace(t *testing.T) {
	host, sources := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	_, errOut, err := executeForTest("lazygit", "--source", " two ")
	if err != nil {
		t.Fatalf("lazygit --source ' two ' error = %v, stderr = %s", err, errOut)
	}
	assertLazygitCall(t, runner, sources["two"])
}

func TestLazygitUnknownSourceFailsBeforeRunning(t *testing.T) {
	host, _ := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	_, _, err := executeForTest("lazygit", "--source", "missing")
	if err == nil {
		t.Fatalf("lazygit error = nil, want unknown source")
	}
	for _, want := range []string{"unknown source \"missing\"", "one", "two"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("lazygit error = %q, want %q", err, want)
		}
	}
	if strings.Index(err.Error(), "one") > strings.Index(err.Error(), "two") {
		t.Fatalf("known sources out of order in error: %q", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestLazygitExplicitEmptySourceFailsBeforeImplicitSelection(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installLazygitFakes(t)

	_, _, err := executeForTest("lazygit", "--source=")
	if err == nil {
		t.Fatalf("lazygit error = nil, want empty source failure")
	}
	if !strings.Contains(err.Error(), "--source") || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("lazygit error = %q, want empty --source", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestLazygitRunnerExitErrorPreservesExitCode(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installLazygitFakes(t)
	runner.err = &exitError{Code: 42, Message: "lazygit exited with status 42"}
	var out, errOut bytes.Buffer

	code := Execute([]string{"lazygit"}, &out, &errOut)

	if code != 42 {
		t.Errorf("Execute(lazygit) = %d, want 42", code)
	}
	if want := "cubby: lazygit exited with status 42\n"; errOut.String() != want {
		t.Errorf("lazygit stderr = %q, want %q", errOut.String(), want)
	}
	if out.Len() != 0 {
		t.Errorf("lazygit stdout = %q, want empty", out.String())
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
	}
}

func TestNormalizeExternalCommandErrorNamesExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is not available on windows")
	}
	runErr := exec.Command("sh", "-c", "exit 42").Run()
	command := externalCommand{Name: "lazygit"}

	err := normalizeExternalCommandError(command, runErr)

	var exitErr *exitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("normalizeExternalCommandError(%v) = %T %v, want exitError", runErr, err, err)
	}
	if exitErr.Code != 42 || exitErr.Message != "lazygit exited with status 42" {
		t.Errorf("normalizeExternalCommandError(%v) = %+v, want code 42 and message %q", runErr, exitErr, "lazygit exited with status 42")
	}
}

func TestLazygitMissingBinaryFailsBeforeTerminalCheck(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)
	t.Setenv("PATH", t.TempDir())
	check := installFakeTerminalCheck(t, errors.New("no terminal"))

	_, _, err := executeForTest("lazygit")

	if err == nil {
		t.Fatalf("lazygit error = nil, want missing binary")
	}
	want := "lazygit not found in PATH; install lazygit or adjust PATH"
	if err.Error() != want {
		t.Fatalf("lazygit error = %q, want %q", err, want)
	}
	if check.calls != 0 {
		t.Fatalf("terminal check calls = %d, want 0", check.calls)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestLazygitWithoutTerminalFailsBeforeRunning(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)
	installFakeLazygitOnPath(t)
	noTerminal := errors.New("lazygit needs a terminal on stdin; run cubby lazygit from an interactive shell, not from a pipe")
	check := installFakeTerminalCheck(t, noTerminal)

	_, _, err := executeForTest("lazygit")

	if !errors.Is(err, noTerminal) {
		t.Fatalf("lazygit error = %v, want %v", err, noTerminal)
	}
	if check.calls != 1 {
		t.Fatalf("terminal check calls = %d, want 1", check.calls)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestLazygitHelpShowsSourceFlag(t *testing.T) {
	out, errOut, err := executeForTest("lazygit", "--help")
	if err != nil {
		t.Fatalf("lazygit --help error = %v, stderr = %s", err, errOut)
	}
	if !strings.Contains(out, "--source") {
		t.Fatalf("lazygit help missing --source:\n%s", out)
	}
}

// installLazygitFakes makes lazygit startable without a real binary or
// terminal: a fake lazygit sits on PATH, the terminal check passes, and the
// returned fake runner records what would have run.
func installLazygitFakes(t *testing.T) *fakeExternalRunner {
	t.Helper()
	installFakeLazygitOnPath(t)
	installFakeTerminalCheck(t, nil)
	return installFakeExternalRunner(t)
}

func installFakeExternalRunner(t *testing.T) *fakeExternalRunner {
	t.Helper()
	runner := &fakeExternalRunner{}
	previous := runExternalCommand
	runExternalCommand = runner.run
	t.Cleanup(func() {
		runExternalCommand = previous
	})
	return runner
}

type fakeTerminalCheck struct {
	calls int
	err   error
}

func (c *fakeTerminalCheck) check() error {
	c.calls++
	return c.err
}

func installFakeTerminalCheck(t *testing.T, err error) *fakeTerminalCheck {
	t.Helper()
	check := &fakeTerminalCheck{err: err}
	previous := requireInteractiveTerminal
	requireInteractiveTerminal = check.check
	t.Cleanup(func() {
		requireInteractiveTerminal = previous
	})
	return check
}

// installFakeLazygitOnPath puts an executable named lazygit on PATH so the
// lookup that precedes the terminal check succeeds. The fake runner means
// it is never executed.
func installFakeLazygitOnPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "lazygit")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	t.Setenv("PATH", dir)
}

func assertLazygitCall(t *testing.T, runner *fakeExternalRunner, wantDir string) {
	t.Helper()
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
	}
	call := runner.calls[0]
	if call.Name != "lazygit" {
		t.Fatalf("command name = %q, want lazygit", call.Name)
	}
	if len(call.Args) != 0 {
		t.Fatalf("command args = %#v, want none", call.Args)
	}
	wantDir = resolvedPath(t, wantDir)
	if call.Dir != wantDir {
		t.Fatalf("command dir = %q, want %q", call.Dir, wantDir)
	}
}

func writeLazygitProject(t *testing.T, sourceNames ...string) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	host := filepath.Join(root, "host")
	sources := make(map[string]string, len(sourceNames))

	hostConfig := ""
	for _, name := range sourceNames {
		sourcePath := filepath.Join(root, name)
		sources[name] = sourcePath
		mustWrite(t, filepath.Join(sourcePath, config.SourceConfigFileName), "profiles = [\"work\"]\n")
		hostConfig += "[[source]]\nname = \"" + name + "\"\npath = \"../" + name + "\"\n\n"
	}
	mustWrite(t, filepath.Join(host, config.HostConfigFileName), hostConfig)
	return host, sources
}
