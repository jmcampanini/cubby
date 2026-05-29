package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/go-config-loader/configloader"
	"github.com/jmcampanini/go-config-loader/configreporter"
	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/spf13/cobra"
)

func configCommand() *cobra.Command {
	var showProvenance bool
	var validatePath string
	var validateSource bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded host config and effective runtime values",
		Long:  "Print the loaded host .cubby.toml after applying defaults, the config file, environment variables, and config-backed flags, followed by commented effective runtime values.",
		Args:  cobra.NoArgs,
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
	cmd.Flags().BoolVar(&validateSource, "source-config", false, "with --validate, validate a source cubby.toml instead of a host .cubby.toml")
	return cmd
}

func writeEffectiveRuntimeComments(out io.Writer, hostRoot string, loadedFiles []string, hostCfg config.HostConfig) error {
	if _, err := fmt.Fprintln(out, "\n# Effective"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# loaded_files = [%s]\n", quotedList(loadedFiles)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# host_root = %q\n", filepath.Clean(hostRoot)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "# effective_profiles = [%s]\n", quotedList(config.EffectiveProfiles(hostCfg))); err != nil {
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

func loadEffectiveHostConfigWithReport(cmd *cobra.Command) (string, config.HostConfig, configloader.LoadReport, error) {
	hostRoot, err := config.CurrentHostRoot()
	if err != nil {
		return "", config.HostConfig{}, configloader.LoadReport{}, err
	}

	hostFile := filepath.Join(hostRoot, config.HostConfigFileName)
	fileLoader, err := configloader.NewRequiredFileLoader[config.HostConfig](hostFile)
	if err != nil {
		return "", config.HostConfig{}, configloader.LoadReport{}, fmt.Errorf("create host config loader for %q: %w", hostFile, err)
	}
	envLoader, err := configloader.NewEnvironmentLoader[config.HostConfig]("cubby", configloader.OSEnv())
	if err != nil {
		return "", config.HostConfig{}, configloader.LoadReport{}, err
	}
	flagLoader, err := pflagloader.NewLoader[config.HostConfig](cmd.Flags())
	if err != nil {
		return "", config.HostConfig{}, configloader.LoadReport{}, err
	}

	hostCfg, report, err := loadHostConfigWithLoaders(hostFile, fileLoader, envLoader, flagLoader)
	if err != nil {
		return "", config.HostConfig{}, configloader.LoadReport{}, err
	}
	return hostRoot, hostCfg, report, nil
}

func loadHostConfigWithLoaders(hostFile string, loaders ...configloader.ConfigLoader[config.HostConfig]) (config.HostConfig, configloader.LoadReport, error) {
	hostCfg, report, err := configloader.Load(config.DefaultHostConfig, loaders...)
	if err != nil {
		return config.HostConfig{}, configloader.LoadReport{}, fmt.Errorf("load host config %q: %w", hostFile, err)
	}
	return config.NormalizeHostConfig(hostCfg), report, nil
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
