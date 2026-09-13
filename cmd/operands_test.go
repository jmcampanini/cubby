package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// TestEveryApplicationCommandDeclaresItsGrammar walks a fresh tree and checks
// the two declarations the CLI contract requires: every application-owned
// command other than the root has an Args validator, and every command with
// subcommands has a runner so Cobra validates operands instead of printing
// help. Cobra validates operands before runners, so the presence of a
// validator is the whole proof that rejected operands reach no command work.
func TestEveryApplicationCommandDeclaresItsGrammar(t *testing.T) {
	root := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{})
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Args == nil {
			t.Errorf("%q has no Args validator", command.CommandPath())
		}
		if command.HasSubCommands() && !command.Runnable() {
			t.Errorf("%q has subcommands but no runner", command.CommandPath())
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	for _, child := range root.Commands() {
		if child.Name() == "help" || child.Name() == "completion" {
			continue
		}
		visit(child)
	}
}
