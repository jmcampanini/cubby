package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func docsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs [manual|schema|reference]",
		Short: "Print Cubby documentation",
		Long:  "Print built-in Cubby documentation for command usage, config schema, or command reference.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most one argument")
			}
			if len(args) == 0 {
				return nil
			}
			switch args[0] {
			case "manual", "schema", "reference":
				return nil
			default:
				return fmt.Errorf("unknown docs topic %q; expected manual, schema, or reference", args[0])
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := "manual"
			if len(args) == 1 {
				topic = args[0]
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), docsContent(topic))
			return err
		},
	}
	return cmd
}

func docsContent(topic string) string {
	switch topic {
	case "schema":
		return strings.TrimSpace(configSchemaDocs) + "\n"
	case "reference":
		return strings.TrimSpace(commandReferenceDocs) + "\n"
	default:
		return strings.TrimSpace(manualDocs) + "\n"
	}
}

const manualDocs = `# Cubby manual

Cubby layers profile-scoped dotfiles from one or more source repositories into a host repository using safe, relative symlinks.

## Workflow

1. Add a host .cubby.toml with selected profiles and registered sources.
2. Add a cubby.toml to each source repository declaring available profiles.
3. Run cubby gitignore sync, cubby link, cubby status, cubby doctor, cubby unlink, or cubby prune as needed.

Use cubby config to inspect the effective host config and provenance. Use cubby docs schema for config file fields and cubby docs reference for command summaries.`

const configSchemaDocs = `# Cubby config schema

## Host .cubby.toml

profiles = ["work"]          # selected profiles
env_profiles = "CUBBY_EXTRA" # optional env var whose comma-separated values are appended
ignore_conflicts = false      # skip conflicting host paths instead of failing link
case_sensitive = false        # treat projected host paths as case-sensitive

[[source]]
name = "dotfiles"            # letters, numbers, underscores, and dashes
path = "../dotfiles"         # absolute, ~/..., or host-root-relative path

## Source cubby.toml

profiles = ["work"]          # profiles declared by this source
ignore = ["**/*.draft.*"]    # optional doublestar patterns to ignore while linking`

const commandReferenceDocs = `# Cubby command reference

cubby link [--dry-run] [--profile PROFILE] [--ignore-conflicts] [--case-sensitive]
    Create managed symlinks for selected profiles.

cubby unlink [--dry-run] [--profile PROFILE]
    Remove managed symlinks for selected profiles.

cubby status [--json]
    Show managed links and drift.

cubby doctor [--json] [--profile PROFILE]
    Check gitignore, sources, requested profiles, dangling links, drift, and conflicts.

cubby prune [--json]
    Remove dangling managed symlinks.

cubby gitignore check|sync [--json]
    Check or append required profile ignore patterns.

cubby profile list|effective [--json] [--profile PROFILE]
    List declared profiles or print the effective profile selection.

cubby source list [--json]
    List registered sources.

cubby lazygit [--source NAME]
    Open lazygit in a registered source repository.

cubby config [--provenance] [--profile PROFILE] [--ignore-conflicts] [--case-sensitive]
    Print the effective host config, optionally with provenance.

cubby config --validate PATH [--source-config]
    Validate a host .cubby.toml or source cubby.toml.

cubby docs [manual|schema|reference]
    Print built-in documentation.

cubby completion SHELL
    Generate shell completion scripts.

cubby --version
    Print the build version.`
