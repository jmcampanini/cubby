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
     help, and 'cubby help <name>' for a name that does not exist, which
     prints 'Unknown help topic' and the root usage on stdout.
  1  Any failure that is not a usage error: a missing or invalid
     .cubby.toml or cubby.toml, a source that cannot be loaded, a
     selected profile no source declares, a filesystem error, lazygit
     not found on PATH, or lazygit asked to start without a terminal.
     Three commands exit 1 silently after writing their report to
     stdout: 'doctor' with at least one issue, 'gitignore check' with at
     least one missing pattern, and 'link' with a fatal conflict, with
     or without --dry-run or --json.
  2  A usage error: an unknown command, flag, or operand, operands that
     do not fit the command's grammar, or a flag value outside its fixed
     set, such as --color=bold or an empty --profile.
  N  'lazygit' exits with lazygit's own exit status once lazygit has
     started, printing nothing of its own. If lazygit is terminated by a
     signal, cubby exits 1 with an error on stderr.

Apart from the three silent exits, every failure prints exactly one line
on stderr in the form 'cubby: <message>' and no usage text. --json never
changes the exit status; the JSON document reports the same result the
exit status does.`,
	)
}
