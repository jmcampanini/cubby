package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/spf13/cobra"
)

type statusEnvelope struct {
	Links []statusLink `json:"links"`
}

type statusLink struct {
	State   string   `json:"state"`
	Path    string   `json:"path"`
	Source  string   `json:"source"`
	Profile string   `json:"profile,omitempty"`
	Target  string   `json:"target"`
	Reasons []string `json:"reasons,omitempty"`
}

func statusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report linked profile files and drift",
		Long: `List managed links under the host root, one per line on stdout, and exit 0
whether or not any link has drifted. Nothing is modified. All registered
sources must load, so a missing source directory is an error here; use
'cubby doctor' or 'cubby prune' in that case.

Each line is '<STATE> <host path> [source=<name> profile=<profile>
target=<source path> reasons=<list>]' where STATE is LINK for a healthy
link or DRIFT when at least one drift reason applies. profile= is omitted
when the basename matches no profile the source declares, and reasons=
appears only on DRIFT lines as a comma-separated list.

` + managedLinkHelp + `

` + driftReasonsHelp + `

` + jsonContractHelp + `
The document is {"links": [...]}, each link carrying "state" ("linked" or
"drift"), "path", "source", "target", and, when present, "profile" and
"reasons".`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}
			links, err := hostlinks.Discover(project.HostRoot, project.Sources)
			if err != nil {
				return err
			}
			jsonOutput, err := jsonOutputEnabled(cmd)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeCommandJSON(cmd, statusEnvelope{Links: statusJSONLinks(links)})
			}
			for _, link := range links {
				if err := renderStatusLine(cmd, link); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "print status as JSON")
	return cmd
}

func statusJSONLinks(links []hostlinks.ManagedLink) []statusLink {
	items := make([]statusLink, 0, len(links))
	for _, link := range links {
		state := "linked"
		if len(link.DriftReasons) > 0 {
			state = "drift"
		}
		items = append(items, statusLink{
			State:   state,
			Path:    slashPath(link.HostRelPath),
			Source:  link.SourceName,
			Profile: link.Profile,
			Target:  slashPath(link.SourceRelPath),
			Reasons: link.DriftReasons,
		})
	}
	return items
}

func renderStatusLine(cmd *cobra.Command, link hostlinks.ManagedLink) error {
	state := "LINK"
	if len(link.DriftReasons) > 0 {
		state = "DRIFT"
	}
	profile := ""
	if link.Profile != "" {
		profile = " profile=" + link.Profile
	}
	reasons := ""
	if len(link.DriftReasons) > 0 {
		reasons = " reasons=" + strings.Join(link.DriftReasons, ",")
	}
	_, err := fmt.Fprintf(commandOut(cmd), "%s %s [source=%s%s target=%s%s]\n",
		state,
		filepath.ToSlash(link.HostRelPath),
		link.SourceName,
		profile,
		filepath.ToSlash(link.SourceRelPath),
		reasons,
	)
	return err
}
