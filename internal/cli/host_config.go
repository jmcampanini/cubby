package cli

import (
	"fmt"
	"path/filepath"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/go-config-loader/configloader"
	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/spf13/cobra"
)

func loadEffectiveHostConfig(cmd *cobra.Command) (string, config.HostConfig, error) {
	hostRoot, hostCfg, _, err := loadEffectiveHostConfigWithReport(cmd)
	if err != nil {
		return "", config.HostConfig{}, err
	}
	return hostRoot, hostCfg, nil
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
