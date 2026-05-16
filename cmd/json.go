package cmd

import (
	"encoding/json"
	"path/filepath"

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

func slashPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}
