package profilefiles

import (
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestSourceSelectedProfilesUsesSourceDeclaredProfiles(t *testing.T) {
	source := config.RegisteredSource{Config: config.SourceConfig{Profiles: []string{"work", "client"}}}
	got := sourceSelectedProfiles(source, []string{"home", "work", "client", "work"})
	assertStringSlicesEqual(t, got, []string{"work", "client"})
}

func TestSourceSelectedProfilesOmittedSourceProfilesSelectNothing(t *testing.T) {
	source := config.RegisteredSource{}
	got := sourceSelectedProfiles(source, []string{"work"})
	assertStringSlicesEqual(t, got, nil)
}
