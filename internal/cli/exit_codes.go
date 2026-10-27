package cli

import "github.com/spf13/cobra"

func exitCodesTopic() *cobra.Command {
	return newHelpTopic(
		"exit-codes",
		"Exit codes and error categories",
		`cubby exits 0, 1, or 2, or passes through the exit status of lazygit:

  0  Success. Reported outcomes that do not count as failures also exit
     0: 'status' listing DRIFT links, 'link' and 'unlink' printing SKIP
     lines, 'profile effective' finding no selected profiles (a notice
     goes to stderr), 'gitignore sync' and 'prune' whether or not they
     changed anything, --help, --version, a bare command group printing
     help, and a valid 'cubby help <name>'.
  1  Any failure that is not a usage error: a missing or invalid
     .cubby.toml or cubby.toml, a source that cannot be loaded, a
     selected profile no source declares, a filesystem error, lazygit
     not found on PATH, or lazygit asked to start without a terminal.
     'doctor' with at least one issue, 'gitignore check' with at least
     one missing pattern, and 'link' with a fatal conflict keep their
     reports on stdout and print a counted failure summary on stderr,
     with or without --dry-run or --json. Examples:
       cubby: doctor found 2 issues
       cubby: .gitignore is missing 1 pattern
       cubby: 1 fatal link conflict
  2  A usage error: an unknown command, help topic, completion shell,
     flag, or operand, operands that do not fit the command's grammar,
     a flag missing its value, or a flag value its flag rejects, such as
     --color=bold or an empty --profile.
  N  'lazygit' exits with lazygit's own exit status once lazygit has
     started. A nonzero status N prints 'cubby: lazygit exited with
     status N' on stderr after lazygit returns. A zero status prints
     nothing from cubby. If lazygit is terminated by a signal, cubby
     exits 1 with an error on stderr.

Every failure ends with exactly one line on stderr in the form
'cubby: <message>' and no usage text. Diagnostics explaining failed
reads or checks precede that final line on stderr. Warnings and notices
also go to stderr. Payloads contain per-item results, including doctor
findings, but no error text or failure records. --json never changes
the exit status or writes a machine-readable error object.`,
	)
}
