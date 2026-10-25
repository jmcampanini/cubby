package output

import "github.com/jmcampanini/cubby/internal/linkops"

// ActionEnvelope is the JSON document link and unlink print.
type ActionEnvelope struct {
	DryRun  bool         `json:"dry_run"`
	Actions []ActionItem `json:"actions"`
}

// ActionItem is one planned action in an ActionEnvelope.
type ActionItem struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Reason string `json:"reason,omitempty"`
	Fatal  bool   `json:"fatal,omitempty"`
}

// LinkActionsEnvelope formats a link or unlink plan for JSON output.
func LinkActionsEnvelope(dryRun bool, actions []linkops.Action) ActionEnvelope {
	items := make([]ActionItem, 0, len(actions))
	for _, action := range actions {
		items = append(items, ActionItem{
			Kind:   string(action.Kind),
			Path:   SlashPath(action.RelPath),
			Source: action.SourceName,
			Target: SlashPath(action.LinkTarget),
			Reason: action.Reason,
			Fatal:  action.Fatal,
		})
	}
	return ActionEnvelope{DryRun: dryRun, Actions: items}
}
