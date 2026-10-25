package config

import (
	"strings"
	"testing"
)

func TestValidateSelectedProfilesRejectsUnknownBeforeSideEffects(t *testing.T) {
	project := &Project{
		Sources: []RegisteredSource{
			{Config: SourceConfig{Profiles: []string{"work", "personal"}}},
		},
	}

	err := ValidateSelectedProfiles(project, []string{"work", "ghost"})
	if err == nil {
		t.Fatalf("ValidateSelectedProfiles() error = nil, want unknown profile")
	}
	if !strings.Contains(err.Error(), "ghost") || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("ValidateSelectedProfiles() error = %q, want unknown profile guidance", err)
	}
}
