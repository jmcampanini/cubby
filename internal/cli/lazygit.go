package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

func lazygitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lazygit",
		Short: "Open lazygit in a source repo",
		Long: `Run lazygit from PATH with its working directory set to a registered source
directory and hand it the terminal: stdin, stdout, and stderr are
inherited, so this command is interactive and prints nothing of its own
while lazygit runs.
With one registered source that source is used; with several, --source
NAME is required. An unknown name is an error listing the known sources,
and an empty name is an error. All sources must load before lazygit
starts.

lazygit is the only external program cubby runs and the only command that
needs an interactive terminal. Before starting it cubby checks, in this
order, that lazygit is on PATH, that stdin is a terminal, that stdout is a
terminal, and that TERM names a terminfo entry with cursor addressing
(TERM=dumb has none). The first missing piece stops the command with
exit status 1 and one line on stderr naming it and how to supply it,
such as 'cubby: lazygit not found in PATH; install lazygit or adjust
PATH' or 'cubby: lazygit needs a terminal on stdout; run cubby lazygit
from an interactive shell, not with stdout redirected'. Configuration and
source errors come before these checks, so a pipe still sees them.
--color does not apply to lazygit's own output. Once lazygit has started,
cubby exits with lazygit's exit status. A nonzero status N prints
'cubby: lazygit exited with status N' on stderr after lazygit returns;
a zero status prints nothing from cubby. If lazygit is terminated by a
signal cubby exits 1 with an error on stderr.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}

			requested, err := cmd.Flags().GetString("source")
			if err != nil {
				return err
			}
			source, err := selectSource(project, requested, cmd.Flags().Changed("source"))
			if err != nil {
				return err
			}

			return runLazygitInSource(source)
		},
	}
	cmd.Flags().String("source", "", "registered source name")
	return cmd
}

func runLazygitInSource(source config.RegisteredSource) error {
	command := externalCommand{Name: "lazygit", Dir: source.ResolvedPath}
	if _, err := exec.LookPath(command.Name); err != nil {
		return normalizeExternalCommandError(command, err)
	}
	if err := requireInteractiveTerminal(); err != nil {
		return err
	}
	if err := runExternalCommand(command); err != nil {
		err = normalizeExternalCommandError(command, err)

		var exitErr *exitError
		if errors.As(err, &exitErr) {
			return err
		}
		return fmt.Errorf("run %s in source %q: %w", command.Name, source.Name, err)
	}
	return nil
}
