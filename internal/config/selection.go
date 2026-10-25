package config

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/output"
)

// ValidateSelectedProfiles checks that selected profiles exist in registered sources.
func ValidateSelectedProfiles(project *Project, profiles []string) error {
	profiles = NormalizeProfiles(profiles)
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles selected; set top-level profiles in %s, CUBBY_PROFILES, --profiles/--profile, or env_profiles", HostConfigFileName)
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
		return fmt.Errorf("selected profiles %s are not declared by any registered source", output.QuotedList(unknown))
	}
	return nil
}

// SourceSelectedProfiles returns selected profiles declared by the source.
func SourceSelectedProfiles(source RegisteredSource, selected []string) []string {
	allowed := make(map[string]struct{})
	for _, profile := range NormalizeProfiles(source.Config.Profiles) {
		allowed[profile] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}

	profiles := make([]string, 0, len(selected))
	for _, profile := range NormalizeProfiles(selected) {
		if _, ok := allowed[profile]; ok {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}
