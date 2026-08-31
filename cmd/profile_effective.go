package cmd

import (
	"fmt"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

type profileEffectiveEnvelope struct {
	Profiles []string `json:"profiles"`
}

func profileEffectiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "effective",
		Short: "Show the effective profile list for the current invocation",
		Long: `Print the profiles the current invocation selects, one per line on stdout,
in selection order. Only .cubby.toml is read: sources are not loaded and
names are not checked against declared profiles, so the command works
while a source is missing and can list a profile nothing declares. When
the list is empty a notice goes to stderr, stdout stays empty, and the
exit status is still 0.

` + profileSelectionHelp + `

` + jsonContractHelp + `
The document is {"profiles": [...]}, with an empty array when nothing is
selected.`,
		Example: `  # with profiles = ["work"] and env_profiles = "CUBBY_EXTRA" in .cubby.toml
  cubby profile effective                          # -> work
  CUBBY_EXTRA=personal cubby profile effective     # -> work, personal
  CUBBY_PROFILES=client cubby profile effective    # -> client
  cubby profile effective --profile client         # -> client`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, hostCfg, err := loadEffectiveHostConfig(cmd)
			if err != nil {
				return err
			}
			profiles := config.EffectiveProfiles(hostCfg)

			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			if jsonOutput {
				if profiles == nil {
					profiles = []string{}
				}
				return writeCommandJSON(cmd, profileEffectiveEnvelope{Profiles: profiles})
			}

			if len(profiles) == 0 {
				_, err := fmt.Fprintln(commandErr(cmd), "no profiles selected; set top-level profiles in .cubby.toml, CUBBY_PROFILES, --profiles/--profile, or env_profiles")
				return err
			}
			for _, p := range profiles {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), p); err != nil {
					return err
				}
			}
			return nil
		},
	}
	addProfileFlag(cmd)
	cmd.Flags().Bool("json", false, "print effective profiles as JSON")
	return cmd
}
