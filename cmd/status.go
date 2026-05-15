package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmcampanini/cubby/internal/config"
	"github.com/jmcampanini/cubby/internal/hostlinks"
	"github.com/spf13/cobra"
)

func statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Report linked profile files and drift",
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := config.LoadProject()
			if err != nil {
				return err
			}
			links, err := hostlinks.Discover(project.HostRoot, project.Sources)
			if err != nil {
				return err
			}
			for _, link := range links {
				if err := renderStatusLine(cmd, link); err != nil {
					return err
				}
			}
			return nil
		},
	}
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
