package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/gitignore"
	"github.com/spf13/cobra"
)

// ExitError asks main to exit with Code without printing an additional error message.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// NewRootCommand builds the cubby command tree.
func NewRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cubby",
		Short:         "Layer profile-scoped dotfiles into a host repo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.AddCommand(
		profileNotImplementedCommand("link", "Create symlinks for selected profiles"),
		profileNotImplementedCommand("unlink", "Remove symlinks for selected profiles"),
		notImplementedCommand("prune", "Remove dangling Cubby symlinks"),
		notImplementedCommand("status", "Report linked profile files and drift"),
		notImplementedCommand("doctor", "Run Cubby health checks"),
		profileCommand(),
		sourceCommand(),
		gitignoreCommand(),
		sourceNotImplementedCommand("lazygit", "Open lazygit in a source repo"),
	)

	return cmd
}

func profileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile commands",
	}
	cmd.AddCommand(notImplementedCommand("list", "List profiles declared by registered sources"))
	return cmd
}

func sourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Source repository commands",
	}
	cmd.AddCommand(notImplementedCommand("list", "List registered source repositories"))
	return cmd
}

func gitignoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitignore",
		Short: "Check or update required host .gitignore patterns",
		Long:  "Check or update the host repository's .gitignore patterns for every profile declared by registered source repos.",
	}
	cmd.AddCommand(gitignoreCheckCommand(), gitignoreSyncCommand())
	return cmd
}

func gitignoreCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Report missing required .gitignore patterns",
		Long:  "Load the host and source Cubby configs, compute required patterns for all declared profiles, and print each missing .gitignore pattern. Exits non-zero when any pattern is missing.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject(".")
			if err != nil {
				return err
			}

			missing, err := missingPatterns(project)
			if err != nil {
				return err
			}
			for _, pattern := range missing {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), pattern); err != nil {
					return err
				}
			}
			if len(missing) > 0 {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
}

func gitignoreSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Append missing required .gitignore patterns",
		Long:  "Load the host and source Cubby configs, compute required patterns for all declared profiles, and append each missing pattern to the host repository's .gitignore.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject(".")
			if err != nil {
				return err
			}

			missing, err := missingPatterns(project)
			if err != nil {
				return err
			}
			gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
			if err := gitignore.AppendMissing(gitignorePath, missing); err != nil {
				return fmt.Errorf("update %s: %w", gitignorePath, err)
			}
			for _, pattern := range missing {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), pattern); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func missingPatterns(project *config.Project) ([]string, error) {
	profiles := project.DeclaredProfiles()
	required := gitignore.RequiredPatterns(profiles)
	gitignorePath := filepath.Join(project.HostRoot, ".gitignore")
	missing, err := gitignore.MissingPatternsFile(gitignorePath, required)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", gitignorePath, err)
	}
	return missing, nil
}

func profileNotImplementedCommand(use, short string) *cobra.Command {
	cmd := notImplementedCommand(use, short)
	cmd.Flags().StringSlice("profile", nil, "profile name (repeatable or comma-separated)")
	return cmd
}

func sourceNotImplementedCommand(use, short string) *cobra.Command {
	cmd := notImplementedCommand(use, short)
	cmd.Flags().String("source", "", "registered source name")
	return cmd
}

func notImplementedCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s: not implemented", cmd.CommandPath())
		},
	}
}
