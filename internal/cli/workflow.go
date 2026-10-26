package cli

import "github.com/spf13/cobra"

func workflowTopic() *cobra.Command {
	return newHelpTopic(
		"workflow",
		"The loop from a fresh host repository to linked dotfiles",
		`cubby keeps profile-scoped dotfiles in source repositories and links the
files of the selected profiles into a host repository. The usual loop:

  1. Register sources. Put a .cubby.toml at the host root naming the
     selected profiles and one [[source]] entry per source repository.
     'cubby config --help' describes every key, and 'cubby config'
     prints the configuration an invocation would use.
  2. Declare profiles. Put a cubby.toml in each source repository naming
     the profiles it provides, and give each profile-scoped file a
     .<profile> name segment as 'cubby link --help' describes.
  3. Keep generated paths out of git. 'cubby gitignore sync' appends the
     patterns the host .gitignore needs.
  4. Link. 'cubby link' creates the symlinks for the selected profiles;
     its help explains how to preview the plan first and how conflicts
     are handled. 'cubby profile effective' shows which profiles apply.
  5. Watch for drift. 'cubby status' lists every managed link, and
     'cubby doctor' checks the whole setup and exits 1 on any issue.
  6. Clean up. 'cubby unlink' removes the links for the selected
     profiles, and 'cubby prune' removes links whose target is gone.

Run every command from the host root; 'cubby --help' explains why.`,
	)
}
