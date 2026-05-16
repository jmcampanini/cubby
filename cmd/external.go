package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type externalCommand struct {
	Name string
	Args []string
	Dir  string
}

type externalCommandRunner func(externalCommand) error

var runExternalCommand externalCommandRunner = productionExternalCommandRunner

var errExternalCommandNotFound = errors.New("external command not found")

type externalCommandNotFoundError struct {
	name string
}

func (e *externalCommandNotFoundError) Error() string {
	return fmt.Sprintf("%s not found in PATH; install %s or adjust PATH", e.name, e.name)
}

func (e *externalCommandNotFoundError) Is(target error) bool {
	return target == errExternalCommandNotFound
}

func productionExternalCommandRunner(command externalCommand) error {
	execCmd := exec.Command(command.Name, command.Args...)
	execCmd.Dir = command.Dir
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return normalizeExternalCommandError(command, err)
	}
	return nil
}

func normalizeExternalCommandError(command externalCommand, err error) error {
	if err == nil {
		return nil
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return err
	}

	if externalCommandNotFound(err) {
		return &externalCommandNotFoundError{name: command.Name}
	}

	var execExitErr *exec.ExitError
	if errors.As(err, &execExitErr) {
		if code := execExitErr.ExitCode(); code >= 0 {
			return &ExitError{Code: code}
		}
	}

	return err
}

func externalCommandNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || errors.Is(err, errExternalCommandNotFound)
}
