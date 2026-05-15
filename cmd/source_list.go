package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/jmcampanini/cubby/internal/config"
	"github.com/spf13/cobra"
)

// SourceListItem is one registered source inventory row.
type SourceListItem struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Profiles []string `json:"profiles"`
}

func sourceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered source repositories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}

			items := sourceListItems(project)
			jsonOutput, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				return encoder.Encode(items)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), renderSourceListTable(items))
			return err
		},
	}
	cmd.Flags().Bool("json", false, "print source inventory as JSON")
	return cmd
}

func sourceListItems(project *config.Project) []SourceListItem {
	items := make([]SourceListItem, 0, len(project.Sources))
	for _, source := range project.Sources {
		profiles := config.NormalizeProfiles(source.Config.Profiles)
		items = append(items, SourceListItem{
			Name:     source.Name,
			Path:     source.ResolvedPath,
			Profiles: profiles,
		})
	}
	return items
}

func renderSourceListTable(items []SourceListItem) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Name, item.Path, strings.Join(item.Profiles, ",")})
	}

	plain := lipgloss.NewStyle()
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(plain).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return plain.Bold(true)
			}
			return plain
		}).
		Headers("NAME", "PATH", "PROFILES").
		Rows(rows...).String()
}
