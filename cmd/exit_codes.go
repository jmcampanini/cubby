package cmd

import "github.com/spf13/cobra"

func exitCodesTopic() *cobra.Command {
	return &cobra.Command{
		Use:   "exit-codes",
		Short: "Exit codes and error categories",
		Long: `cubby exits 0, exits 1, or passes through the exit status of lazygit:

  0  Success. Reported outcomes that do not count as failures also exit
     0: 'status' listing DRIFT links, 'link' and 'unlink' printing SKIP
     lines, 'profile effective' finding no selected profiles (a notice
     goes to stderr), 'gitignore sync' and 'prune' whether or not they
     changed anything, and --help, --version, or a bare command group
     printing help.
  1  Any failure. Usage errors (an unknown command, operand, flag, or
     docs topic), a missing or invalid .cubby.toml or cubby.toml, a
     source that cannot be loaded, filesystem errors, and lazygit not
     found on PATH print 'cubby: error: <message>' on stderr. Three
     commands exit 1 silently after writing their report to stdout:
     'doctor' with at least one issue, 'gitignore check' with at least
     one missing pattern, and 'link' with a fatal conflict, with or
     without --dry-run or --json.
  N  'lazygit' exits with lazygit's own exit status once lazygit has
     started, printing nothing of its own. If lazygit is terminated by a
     signal, cubby exits 1 with an error on stderr.

--json never changes the exit status; the JSON document reports the same
result the exit status does.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}
