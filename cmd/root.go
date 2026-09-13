// Package cmd defines Cubby's CLI commands and output contracts.
package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the cubby command tree.
func NewRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cubby",
		Short: "Layer profile-scoped dotfiles into a host repo",
		Long: `Layer profile-scoped dotfiles from one or more source repositories into a
host repository as relative symlinks.

A host repository registers its sources in .cubby.toml. Each source
declares its profiles in cubby.toml and keeps profile-scoped files such as
zshrc.work or nvim/init.work.lua. 'cubby link' symlinks the files for the
selected profiles into the host at the same relative paths, 'cubby unlink'
removes those links, 'cubby status' and 'cubby doctor' report on them, and
'cubby prune' removes links whose target is gone. 'cubby gitignore sync'
adds the ignore patterns the host needs for .cubby.toml and linked files.

` + hostRootHelp + `

No command prompts. Inventory, action, and diagnostic commands accept
--json; each command's help states its output contract. Only 'cubby
lazygit' runs an external program: it starts lazygit from PATH in a source
directory and hands it the terminal. cubby never accesses the network.

Run 'cubby config --help' for configuration precedence and the file
schema, 'cubby profile effective' for the profiles an invocation would
use, and 'cubby help exit-codes' for exit-status meanings. 'cubby docs'
prints longer manual, schema, and reference text.`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	// Register --help and --version before Find strips flags from the
	// arguments. Cobra otherwise adds them during execute, after Find has
	// already treated an unregistered --help or --version as taking the next
	// argument as its value.
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	cmd.AddCommand(
		linkCommand(),
		unlinkCommand(),
		pruneCommand(),
		statusCommand(),
		doctorCommand(),
		profileCommand(),
		sourceCommand(),
		gitignoreCommand(),
		lazygitCommand(),
		configCommand(),
		docsCommand(),
		exitCodesTopic(),
	)

	return cmd
}
