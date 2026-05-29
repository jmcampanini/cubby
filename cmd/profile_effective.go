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
