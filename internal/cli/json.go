package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func writeCommandJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(commandOut(cmd))
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func jsonOutputEnabled(cmd *cobra.Command) (bool, error) {
	return cmd.Flags().GetBool("json")
}
