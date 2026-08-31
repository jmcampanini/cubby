package cmd

import (
	"errors"
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

func lazygitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lazygit",
		Short: "Open lazygit in a source repo",
		Long: `Run lazygit from PATH with its working directory set to a registered source
directory and hand it the terminal: stdin, stdout, and stderr are
inherited, so this command is interactive and prints nothing of its own.
With one registered source that source is used; with several, --source
NAME is required. An unknown name is an error listing the known sources,
and an empty name is an error. All sources must load before lazygit
starts.

lazygit is the only external program cubby runs. When it is not on PATH
the command fails with 'lazygit not found in PATH; install lazygit or
adjust PATH' and exit status 1 without starting anything. Once lazygit has
started, cubby exits with lazygit's exit status; if lazygit is terminated
by a signal cubby exits 1 with an error on stderr.`,
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
	if err := runExternalCommand(command); err != nil {
		err = normalizeExternalCommandError(command, err)

		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			return err
		}
		if externalCommandNotFound(err) {
			return err
		}
		return fmt.Errorf("run %s in source %q: %w", command.Name, source.Name, err)
	}
	return nil
}
