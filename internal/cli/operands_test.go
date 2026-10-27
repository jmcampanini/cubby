package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// TestEveryApplicationCommandDeclaresItsGrammar walks a fresh tree and checks
// the declarations the CLI contract requires: every command other than the
// root, Cobra's help and completion included, has an Args validator, and
// every command group and help topic has a runner so Cobra validates
// operands instead of printing help. Cobra validates operands before runners, so the presence of
// a validator is the whole proof that rejected operands reach no command
// work.
func TestEveryApplicationCommandDeclaresItsGrammar(t *testing.T) {
	root := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{})

	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Args == nil {
			t.Errorf("%q has no Args validator", command.CommandPath())
		}
		if command.HasSubCommands() && !command.Runnable() {
			t.Errorf("%q has subcommands but no runner", command.CommandPath())
		}
		if command.GroupID == helpTopicGroupID && !command.Runnable() {
			t.Errorf("%q is a help topic but has no runner", command.CommandPath())
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	for _, child := range root.Commands() {
		visit(child)
	}
}
