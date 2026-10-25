package cli

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/spf13/cobra"
)

func addProfileFlag(cmd *cobra.Command) {
	if err := pflagloader.Register[config.HostConfig](cmd.Flags()); err != nil {
		panic(err)
	}
}

func loadProfileScopedProject(cmd *cobra.Command) (*config.Project, []string, error) {
	hostRoot, hostCfg, err := loadEffectiveHostConfig(cmd)
	if err != nil {
		return nil, nil, err
	}

	project, err := config.LoadProjectWithHostConfig(hostRoot, hostCfg)
	if err != nil {
		return nil, nil, err
	}

	profiles := config.EffectiveProfiles(project.Host)
	if err := config.ValidateSelectedProfiles(project, profiles); err != nil {
		return nil, nil, err
	}
	return project, profiles, nil
}

func renderMissingProfileDiagnostics(cmd *cobra.Command, project *config.Project, profiles []string) error {
	profiles = config.NormalizeProfiles(profiles)
	for _, source := range project.Sources {
		allowed := make(map[string]struct{}, len(source.Config.Profiles))
		for _, profile := range config.NormalizeProfiles(source.Config.Profiles) {
			allowed[profile] = struct{}{}
		}
		for _, profile := range profiles {
			if _, ok := allowed[profile]; ok {
				continue
			}
			if _, err := fmt.Fprintf(commandErr(cmd), "source %q does not declare selected profile %q; skipping\n", source.Name, profile); err != nil {
				return err
			}
		}
	}
	return nil
}
