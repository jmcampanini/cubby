package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/gitignore"
	"github.com/spf13/cobra"
)

type gitignoreSyncEnvelope struct {
	Changed bool     `json:"changed"`
	Added   []string `json:"added"`
}

func gitignoreSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Append missing required .gitignore patterns",
		Long: `Append each missing required pattern to the host .gitignore, creating the
file when absent, and print each appended pattern on stdout, one per line.
Existing lines are never removed, reordered, or rewritten; when the file
lacks a trailing newline one is added before the new patterns. Exit
status is 0 whether or not anything was appended.

` + gitignorePatternsHelp + `

` + jsonContractHelp + `
The document is {"changed": bool, "added": [...]}.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
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
			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeCommandJSON(cmd, gitignoreSyncEnvelope{Changed: len(missing) > 0, Added: missing})
			}
			for _, pattern := range missing {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), pattern); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "print .gitignore sync result as JSON")
	return cmd
}
