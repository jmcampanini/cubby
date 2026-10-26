package cli

import "github.com/spf13/cobra"

func workflowTopic() *cobra.Command {
	return newHelpTopic(
		"workflow",
		"The loop from a fresh host repository to linked dotfiles",
		`cubby keeps profile-scoped dotfiles in source repositories and links the
files of the selected profiles into a host repository. The usual loop:

  1. Register sources. Put a .cubby.toml at the host root naming the
     selected profiles and one [[source]] entry per source repository;
     'cubby config --help' describes every key.
  2. Declare profiles. Put a cubby.toml in each source repository naming
     the profiles it provides, and give each profile-scoped file a
     .<profile> name segment; 'cubby link --help' describes the grammar.
  3. Keep generated paths out of git with 'cubby gitignore sync'.
  4. Link the selected profiles with 'cubby link'; 'cubby profile
     effective' shows which profiles that is.
  5. Watch for drift with 'cubby status' and 'cubby doctor'.
  6. Clean up with 'cubby unlink' for selected profiles and 'cubby prune'
     for links whose target is gone.

Each command's --help states its contract. Run every command from the
host root; 'cubby --help' explains why.`,
	)
}
