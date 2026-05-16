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
		Args:  cobra.NoArgs,
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
