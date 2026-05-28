package cmd

import (
	"fmt"
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
		Short: "Print the effective Cubby host config",
		Long:  "Print the effective host .cubby.toml after applying defaults, the config file, environment variables, and config-backed flags.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if validatePath != "" {
				return validateConfigFile(cmd, validatePath, validateSource)
			}

			_, hostCfg, report, err := loadEffectiveHostConfigReport(cmd)
			if err != nil {
				return err
			}

			reporter := configreporter.New(hostCfg, report)
			if err := reporter.WriteTOML(cmd.OutOrStdout()); err != nil {
				return err
			}
			if !showProvenance {
				return nil
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "\n# Provenance"); err != nil {
				return err
			}
			return writeProvenanceTable(cmd, reporter)
		},
	}
	addProfileFlag(cmd)
	cmd.Flags().BoolVar(&showProvenance, "provenance", true, "include config provenance")
	cmd.Flags().StringVar(&validatePath, "validate", "", "validate a config file and exit")
	cmd.Flags().BoolVar(&validateSource, "source-config", false, "with --validate, validate a source cubby.toml instead of a host .cubby.toml")
	return cmd
}

func validateConfigFile(cmd *cobra.Command, path string, source bool) error {
	if source {
		if _, err := config.LoadSourceConfigFile(path, "source"); err != nil {
			return err
		}
	} else if _, err := config.LoadHostConfigFile(path); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "valid")
	return err
}

func loadEffectiveHostConfigReport(cmd *cobra.Command) (string, config.HostConfig, configloader.LoadReport, error) {
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

func writeProvenanceTable(cmd *cobra.Command, reporter configreporter.Reporter[config.HostConfig]) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	headers := reporter.ProvenanceHeaders()
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", headers[0], headers[1], headers[2]); err != nil {
		return err
	}
	for _, row := range reporter.ProvenanceRows() {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", row[0], row[1], row[2]); err != nil {
			return err
		}
	}
	return w.Flush()
}
