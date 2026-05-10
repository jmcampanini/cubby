package cmd

import "testing"

func TestSelectedProfilesRequiresNonEmptyProfileFlag(t *testing.T) {
	cmd := linkCommand()
	cmd.SetArgs([]string{"--profile", "  "})
	if err := cmd.ParseFlags([]string{"--profile", "  "}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if _, err := selectedProfiles(cmd); err == nil {
		t.Fatalf("selectedProfiles() error = nil, want required profile error")
	}
}

func TestSelectedProfilesTrimsAndDeduplicates(t *testing.T) {
	cmd := linkCommand()
	if err := cmd.ParseFlags([]string{"--profile", " work ", "--profile", "work"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	profiles, err := selectedProfiles(cmd)
	if err != nil {
		t.Fatalf("selectedProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0] != "work" {
		t.Fatalf("selectedProfiles() = %#v, want [work]", profiles)
	}
}
