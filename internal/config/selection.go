package config

import (
	"fmt"
	"strings"
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
			unknown = append(unknown, fmt.Sprintf("%q", profile))
		}
	}
	if len(unknown) > 0 {
		if len(unknown) == 1 {
			return fmt.Errorf("selected profile %s is not declared by any registered source", unknown[0])
		}
		return fmt.Errorf("selected profiles %s are not declared by any registered source", strings.Join(unknown, ", "))
	}
	return nil
}
