package cmd

import (
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestSelectedProfilesRequiresNonEmptyProfileFlag(t *testing.T) {
	cmd := linkCommand()
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
	assertProfilesEqual(t, profiles, []string{"work"})
}

func TestSourceSelectedProfilesRequiresHostOptIn(t *testing.T) {
	source := config.RegisteredSource{HostSource: config.HostSource{Profiles: []string{"work", "client"}}}
	got := sourceSelectedProfiles(source, []string{"home", "work", "client", "work"})
	assertProfilesEqual(t, got, []string{"work", "client"})
}

func TestSourceSelectedProfilesOmittedHostProfilesSelectNothing(t *testing.T) {
	source := config.RegisteredSource{}
	got := sourceSelectedProfiles(source, []string{"work"})
	assertProfilesEqual(t, got, nil)
}

func assertProfilesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("profiles = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("profiles = %#v, want %#v", got, want)
		}
	}
}
