package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

func addProfileFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("profile", nil, "profile name (repeatable or comma-separated)")
}

func selectedProfiles(cmd *cobra.Command) ([]string, error) {
	values, err := cmd.Flags().GetStringSlice("profile")
	if err != nil {
		return nil, err
	}

	profiles := normalizeProfiles(values)
	if len(profiles) == 0 {
		return nil, fmt.Errorf("--profile is required")
	}
	return profiles, nil
}

func sourceSelectedProfiles(source config.RegisteredSource, selected []string) []string {
	allowed := make(map[string]struct{})
	for _, profile := range normalizeProfiles(source.Profiles) {
		allowed[profile] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}

	profiles := make([]string, 0, len(selected))
	for _, profile := range normalizeProfiles(selected) {
		if _, ok := allowed[profile]; ok {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func normalizeProfiles(values []string) []string {
	profiles := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		profile := strings.TrimSpace(value)
		if profile == "" {
			continue
		}
		if _, ok := seen[profile]; ok {
			continue
		}
		seen[profile] = struct{}{}
		profiles = append(profiles, profile)
	}
	return profiles
}
