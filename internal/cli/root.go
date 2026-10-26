// Package cli defines Cubby's CLI commands and output contracts.
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// Execute runs the command tree on args and returns the process exit status:
// 0 on success, 2 on a usage error, 1 on any other failure unless the error
// is an exitError carrying its own status.
func Execute(args []string, out, errOut io.Writer) int {
	cmd := newRootCommand(out, errOut)
	cmd.SetArgs(args)

	// Cobra resolves the command, parses flags, and validates Args before it
	// runs this hook, so an error returned while the hook has not fired is a
	// usage error. Cobra validates required flags and flag groups after the
	// hook, and runs only the nearest PersistentPreRun in the tree, so no
	// command declares required flags, flag groups, or its own hook.
	runnerReached := false
	cmd.PersistentPreRun = func(*cobra.Command, []string) { runnerReached = true }

	err := cmd.Execute()
	if err == nil {
		return 0
	}
	var exitErr *exitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	// The diagnostic is one line, so a message that carries newlines, such
	// as a flag name or path typed with one, is joined with spaces.
	_, _ = fmt.Fprintf(errOut, "cubby: %s\n", strings.ReplaceAll(err.Error(), "\n", " "))
	if runnerReached {
		return 1
	}
	return 2
}

// newRootCommand builds the cubby command tree.
func newRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cubby",
		Short: "Layer profile-scoped dotfiles into a host repo",
		Long: `Layer profile-scoped dotfiles from one or more source repositories into a
host repository as relative symlinks.

A host repository registers its sources in .cubby.toml. Each source
declares its profiles in cubby.toml and keeps profile-scoped files such as
zshrc.work or nvim/init.work.lua. cubby links the files of the selected
profiles into the host at the same relative paths, reports on those
links, and removes them again. It never replaces an existing host file
and never accesses the network.

` + hostRootHelp + `

No command prompts. Inventory, action, and diagnostic commands accept
--json; each command's help states its output contract. The only styled
output cubby renders is the 'source list' table header. Only 'cubby
lazygit' runs an external program: it starts lazygit from PATH in a
source directory and hands it the terminal, so it is the one command
that needs an interactive terminal. Help topics are read with
'cubby help <topic>'.

` + colorModeHelp + `

Common tasks:
  Cover generated paths in .gitignore:  cubby gitignore sync
  Link the selected profiles:           cubby link
  List links and their drift:           cubby status
  Check the whole setup:                cubby doctor

Command groups:
  profile    Declared and selected profiles:     cubby profile --help
  source     Registered source repositories:     cubby source --help
  gitignore  Required host .gitignore patterns:  cubby gitignore --help`,
		Version:            Version,
		DisableSuggestions: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	// Register --help and --version before Find strips flags from the
	// arguments. Cobra otherwise adds them during execute, after Find has
	// already treated an unregistered --help or --version as taking the next
	// argument as its value.
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()
	addColorFlag(cmd)

	cmd.AddGroup(
		&cobra.Group{ID: commandGroupID, Title: "Commands:"},
		&cobra.Group{ID: helpTopicGroupID, Title: "Help topics:"},
	)
	cmd.SetHelpCommandGroupID(commandGroupID)
	cmd.SetCompletionCommandGroupID(commandGroupID)
	for _, command := range []*cobra.Command{
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
	} {
		command.GroupID = commandGroupID
		cmd.AddCommand(command)
	}
	cmd.AddCommand(
		workflowTopic(),
		exitCodesTopic(),
	)

	return cmd
}
