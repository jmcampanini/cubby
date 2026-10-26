package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/output"
	"github.com/jmcampanini/go-config-loader/configreporter"
	"github.com/spf13/cobra"
)

func configCommand() *cobra.Command {
	var showProvenance bool
	var validatePath string
	var validateSource bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded host config and effective runtime values",
		Long: `Print the host configuration as TOML on stdout after applying every layer,
followed by commented effective values, or validate one config file with
--validate. Nothing is modified.

Settings load in this order, and a later layer replaces any value an
earlier one sets: built-in defaults (every field empty or false),
.cubby.toml in the current directory, the environment variables
CUBBY_PROFILES, CUBBY_IGNORE_CONFLICTS, and CUBBY_CASE_SENSITIVE, then the
flags --profiles, --profile, --ignore-conflicts, and --case-sensitive on
the commands that accept them. env_profiles and [[source]] entries are
read only from the file.

` + profileSelectionHelp + `

` + hostRootHelp + `

Host .cubby.toml keys:
  profiles          Profile names to select; may be empty.
  env_profiles      Name of an environment variable whose comma-separated
                    value is appended to the selection when it is set.
  ignore_conflicts  Bool, default false: turn link conflicts into SKIP
                    lines instead of failures.
  case_sensitive    Bool, default false: treat host paths that differ only
                    by letter case as distinct.
  [[source]]        One table per source repository, with keys:
    name            Letters, digits, underscores, and dashes; unique
                    within the host.
    path            Absolute, ~/..., or relative to the host root; a
                    directory holding a cubby.toml. Commands that load
                    sources fail when it is missing.

Source cubby.toml keys:
  profiles          Profile names the source provides; at least one.
  ignore            Optional doublestar patterns for files to skip while
                    linking: a pattern without '/' matches basenames
                    anywhere in the source, one with '/' matches the
                    source-relative path.

A key either file does not define is an error.

The report is valid TOML that reloads as .cubby.toml. Its profiles line
shows the selected list before env_profiles is applied; the '# Effective'
comment block lists loaded_files, host_root, and effective_profiles with
env_profiles applied. --provenance appends a '# Provenance' table naming
the source of each field: <default>, the file path, <env>, or <pflag>.
Nothing is redacted; host configuration holds no secret fields. Only the
host file is read, so the report works while a source is missing.

--validate PATH loads PATH as a host .cubby.toml, resolves its [[source]]
paths relative to PATH's directory, loads each source's cubby.toml, and
prints 'valid' on stdout; with --source-config it validates PATH as a
source cubby.toml instead. --validate does not read the current directory
and ignores the other flags. Validation errors go to stderr with exit
status 1.

` + noPromptHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if validatePath != "" {
				return validateConfigFile(cmd, validatePath, validateSource)
			}

			hostRoot, hostCfg, report, err := loadEffectiveHostConfigWithReport(cmd)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			reporter := configreporter.New(hostCfg, report)
			if err := reporter.WriteTOML(out); err != nil {
				return err
			}
			if err := writeEffectiveRuntimeComments(out, hostRoot, report.LoadedFiles, hostCfg); err != nil {
				return err
			}
			if !showProvenance {
				return nil
			}

			if _, err := fmt.Fprintln(out, "\n# Provenance"); err != nil {
				return err
			}
			return writeProvenanceTable(out, reporter)
		},
	}
	addProfileFlag(cmd)
	cmd.Flags().BoolVar(&showProvenance, "provenance", false, "include config provenance")
	cmd.Flags().StringVar(&validatePath, "validate", "", "validate a config file and exit")
	cmd.Flags().BoolVar(&validateSource, "source-config", false, "with --validate, treat PATH as a source cubby.toml")
	return cmd
}

func writeEffectiveRuntimeComments(out io.Writer, hostRoot string, loadedFiles []string, hostCfg config.HostConfig) error {
	if _, err := fmt.Fprintln(out, "\n# Effective"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# loaded_files = [%s]\n", output.QuotedList(loadedFiles)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# host_root = %q\n", filepath.Clean(hostRoot)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# effective_profiles = [%s]\n", output.QuotedList(config.EffectiveProfiles(hostCfg))); err != nil {
		return err
	}
	return nil
}

func validateConfigFile(cmd *cobra.Command, path string, source bool) error {
	if source {
		if _, err := config.LoadSourceConfigFile(path, "source"); err != nil {
			return err
		}
	} else if err := validateHostConfigFile(path); err != nil {
		return err
	}

	_, err := fmt.Fprintln(cmd.OutOrStdout(), "valid")
	return err
}

func validateHostConfigFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config path %q: %w", path, err)
	}
	hostCfg, err := config.LoadHostConfigFile(absPath)
	if err != nil {
		return err
	}
	_, err = config.LoadProjectWithHostConfig(filepath.Dir(absPath), hostCfg)
	return err
}

func writeProvenanceTable(out io.Writer, reporter configreporter.Reporter[config.HostConfig]) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	headers := reporter.ProvenanceHeaders()
	if _, err := fmt.Fprintf(w, "# %s\t%s\t%s\n", headers[0], headers[1], headers[2]); err != nil {
		return err
	}
	for _, row := range reporter.ProvenanceRows() {
		if _, err := fmt.Fprintf(w, "# %s\t%s\t%s\n", row[0], row[1], row[2]); err != nil {
			return err
		}
	}
	return w.Flush()
}
