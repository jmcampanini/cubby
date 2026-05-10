package cmd

import (
	"fmt"
	"strings"

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
	if len(profiles) == 0 {
		return nil, fmt.Errorf("--profile is required")
	}
	return profiles, nil
}
