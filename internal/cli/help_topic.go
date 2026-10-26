package cli

import "github.com/spf13/cobra"

// Root help lists commands and help topics under separate headings. The
// group IDs are also how tests tell a help topic from a command.
const (
	commandGroupID   = "commands"
	helpTopicGroupID = "topics"
)

const helpTopicHelp = `This topic prints help on stdout, reads no configuration, and works
without a terminal.`

// newHelpTopic builds a help topic: a command that does no work and prints
// its own help whether invoked directly or through 'cubby help <topic>'.
func newHelpTopic(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long + "\n\n" + helpTopicHelp,
		GroupID: helpTopicGroupID,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}
