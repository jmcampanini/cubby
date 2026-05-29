package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

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
	if err := validateSelectedProfiles(project, profiles); err != nil {
		return nil, nil, err
	}
	return project, profiles, nil
}

func loadEffectiveHostConfig(cmd *cobra.Command) (string, config.HostConfig, error) {
	hostRoot, hostCfg, _, err := loadEffectiveHostConfigWithReport(cmd)
	if err != nil {
		return "", config.HostConfig{}, err
	}
	return hostRoot, hostCfg, nil
}

func validateSelectedProfiles(project *config.Project, profiles []string) error {
	profiles = config.NormalizeProfiles(profiles)
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles selected; set top-level profiles in %s, CUBBY_PROFILES, --profiles/--profile, or env_profiles", config.HostConfigFileName)
	}

	declared := make(map[string]struct{})
	for _, profile := range project.DeclaredProfiles() {
		declared[profile] = struct{}{}
	}

	unknown := make([]string, 0)
	for _, profile := range profiles {
		if _, ok := declared[profile]; !ok {
			unknown = append(unknown, profile)
		}
	}
	if len(unknown) > 0 {
		if len(unknown) == 1 {
			return fmt.Errorf("selected profile %q is not declared by any registered source", unknown[0])
		}
		return fmt.Errorf("selected profiles %s are not declared by any registered source", quotedList(unknown))
	}
	return nil
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

func commandErr(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stderr
	}
	return cmd.ErrOrStderr()
}

func sourceSelectedProfiles(source config.RegisteredSource, selected []string) []string {
	allowed := make(map[string]struct{})
	for _, profile := range config.NormalizeProfiles(source.Config.Profiles) {
		allowed[profile] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}

	profiles := make([]string, 0, len(selected))
	for _, profile := range config.NormalizeProfiles(selected) {
		if _, ok := allowed[profile]; ok {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func quotedList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}
