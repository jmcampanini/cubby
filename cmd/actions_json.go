package cmd

import "github.com/jmcampanini/cubby/internal/linkops"

type actionEnvelope struct {
	DryRun  bool         `json:"dry_run"`
	Actions []actionItem `json:"actions"`
}

type actionItem struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Reason string `json:"reason,omitempty"`
	Fatal  bool   `json:"fatal,omitempty"`
}

func linkActionsEnvelope(dryRun bool, actions []linkops.Action) actionEnvelope {
	items := make([]actionItem, 0, len(actions))
	for _, action := range actions {
		items = append(items, actionItem{
			Kind:   string(action.Kind),
			Path:   slashPath(action.RelPath),
			Source: action.SourceName,
			Target: slashPath(action.LinkTarget),
			Reason: action.Reason,
			Fatal:  action.Fatal,
		})
	}
	return actionEnvelope{DryRun: dryRun, Actions: items}
}
