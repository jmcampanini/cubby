package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/output"
	"github.com/spf13/cobra"
)

// SourceListItem is one registered source inventory row.
type SourceListItem struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Profiles []string `json:"profiles"`
}

type sourceListEnvelope struct {
	Sources []SourceListItem `json:"sources"`
}

func sourceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered source repositories",
		Long: `Print a table of registered sources on stdout with columns NAME, PATH (the
absolute path after expanding ~ and host-relative paths), and PROFILES
(declared, comma-separated), in registration order. All sources must
load. The table uses Unicode box borders; its header is bold only when
--color styles stdout. Use --json for machine reading.

` + colorModeHelp + `

` + jsonContractHelp + `
The document is {"sources": [{"name", "path", "profiles"}]}.

` + noPromptHelp,
		Args: cobra.NoArgs,
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
				return writeCommandJSON(cmd, sourceListEnvelope{Sources: sourceListJSONItems(items)})
			}

			out := cmd.OutOrStdout()
			_, err = fmt.Fprintln(out, renderSourceListTable(items, styledOutput(cmd, out)))
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

func sourceListJSONItems(items []SourceListItem) []SourceListItem {
	jsonItems := make([]SourceListItem, 0, len(items))
	for _, item := range items {
		item.Path = output.SlashPath(item.Path)
		jsonItems = append(jsonItems, item)
	}
	return jsonItems
}

func renderSourceListTable(items []SourceListItem, styled bool) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Name, item.Path, strings.Join(item.Profiles, ",")})
	}

	plain := lipgloss.NewStyle()
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(plain).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow && styled {
				return plain.Bold(true)
			}
			return plain
		}).
		Headers("NAME", "PATH", "PROFILES").
		Rows(rows...).String()
}
