package cmd

import (
	"errors"
	"os/exec"
	"path/filepath"
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
	runner := installFakeExternalRunner(t)

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
	runner := installFakeExternalRunner(t)

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
	runner := installFakeExternalRunner(t)

	_, errOut, err := executeForTest("lazygit", "--source", "two")
	if err != nil {
		t.Fatalf("lazygit --source two error = %v, stderr = %s", err, errOut)
	}
	assertLazygitCall(t, runner, sources["two"])
}

func TestLazygitSourceFlagTrimsWhitespace(t *testing.T) {
	host, sources := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)

	_, errOut, err := executeForTest("lazygit", "--source", " two ")
	if err != nil {
		t.Fatalf("lazygit --source ' two ' error = %v, stderr = %s", err, errOut)
	}
	assertLazygitCall(t, runner, sources["two"])
}

func TestLazygitUnknownSourceFailsBeforeRunning(t *testing.T) {
	host, _ := writeLazygitProject(t, "one", "two")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)

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
	runner := installFakeExternalRunner(t)

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

func TestLazygitRunnerMissingBinaryReturnsFriendlyError(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)
	runner.err = exec.ErrNotFound

	_, _, err := executeForTest("lazygit")
	if err == nil {
		t.Fatalf("lazygit error = nil, want missing binary")
	}
	want := "lazygit not found in PATH; install lazygit or adjust PATH"
	if err.Error() != want {
		t.Fatalf("lazygit error = %q, want %q", err, want)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
	}
}

func TestLazygitRunnerExitErrorPreservesExitCode(t *testing.T) {
	host, _ := writeLazygitProject(t, "one")
	mustChdir(t, host)
	runner := installFakeExternalRunner(t)
	runner.err = &ExitError{Code: 42}

	_, _, err := executeForTest("lazygit")
	if err == nil {
		t.Fatalf("lazygit error = nil, want ExitError")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("lazygit error = %T %v, want ExitError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("ExitError code = %d, want 42", exitErr.Code)
	}
	if err.Error() != "exit status 42" {
		t.Fatalf("lazygit error = %q, want unwrapped exit status", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
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
