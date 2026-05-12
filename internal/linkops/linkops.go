package linkops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ActionKind is the kind of filesystem operation or classification in a plan.
type ActionKind string

const (
	ActionCreate   ActionKind = "create"
	ActionRemove   ActionKind = "remove"
	ActionNoop     ActionKind = "noop"
	ActionSkip     ActionKind = "skip"
	ActionConflict ActionKind = "conflict"
)

// Action is one planned link or unlink classification.
type Action struct {
	Kind       ActionKind
	SourceName string
	SourceRoot string
	SourcePath string
	HostPath   string
	RelPath    string
	LinkTarget string
	Reason     string
	Fatal      bool
}

// Plan is a complete set of actions collected before filesystem mutation.
type Plan struct {
	Actions []Action
}

// HasFatalConflicts reports whether any action is a fatal conflict.
func (p Plan) HasFatalConflicts() bool {
	for _, action := range p.Actions {
		if action.Kind == ActionConflict && action.Fatal {
			return true
		}
	}
	return false
}

// FatalConflicts returns all fatal conflict actions.
func (p Plan) FatalConflicts() []Action {
	conflicts := make([]Action, 0)
	for _, action := range p.Actions {
		if action.Kind == ActionConflict && action.Fatal {
			conflicts = append(conflicts, action)
		}
	}
	return conflicts
}

// Skips returns all skip actions.
func (p Plan) Skips() []Action {
	skips := make([]Action, 0)
	for _, action := range p.Actions {
		if action.Kind == ActionSkip {
			skips = append(skips, action)
		}
	}
	return skips
}

// SourceFiles describes selected profile files discovered for one source.
type SourceFiles struct {
	Name     string
	Root     string
	RelPaths []string
}

// PlanOptions controls conflict classification.
type PlanOptions struct {
	IgnoreConflicts bool
}

// PlanLink classifies all selected source files for link without mutating the filesystem.
func PlanLink(hostRoot string, sources []SourceFiles, opts PlanOptions) (Plan, error) {
	planner := linkPlanner{
		hostRoot:        filepath.Clean(hostRoot),
		ignoreConflicts: opts.IgnoreConflicts,
		seen:            make(map[string]Action),
	}
	for _, source := range sources {
		for _, relPath := range source.RelPaths {
			action, err := planner.planLinkAction(source, relPath)
			if err != nil {
				return Plan{}, err
			}
			planner.actions = append(planner.actions, action)
		}
	}
	return Plan{Actions: planner.actions}, nil
}

type linkPlanner struct {
	hostRoot        string
	ignoreConflicts bool
	seen            map[string]Action
	actions         []Action
}

func (p *linkPlanner) planLinkAction(source SourceFiles, relPath string) (Action, error) {
	action := baseAction(p.hostRoot, source, relPath)
	key := filepath.Clean(relPath)
	if winner, ok := p.seen[key]; ok {
		action.Reason = fmt.Sprintf("host path collision with source %q", winner.SourceName)
		return p.conflictOrSkip(action), nil
	}
	p.seen[key] = action

	info, err := os.Lstat(action.HostPath)
	if err != nil {
		if os.IsNotExist(err) {
			target, err := RelativeTarget(action.HostPath, action.SourcePath)
			if err != nil {
				return Action{}, err
			}
			action.Kind = ActionCreate
			action.LinkTarget = target
			return action, nil
		}
		return Action{}, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		ok, err := PointsTo(action.HostPath, action.SourcePath)
		if err != nil {
			return Action{}, err
		}
		if ok {
			action.Kind = ActionNoop
			action.Reason = "already linked"
			return action, nil
		}
		action.Reason = "unexpected symlink"
		return p.conflictOrSkip(action), nil
	}

	if info.IsDir() {
		action.Reason = "host path is a directory"
	} else {
		action.Reason = "host path already exists"
	}
	return p.conflictOrSkip(action), nil
}

func (p *linkPlanner) conflictOrSkip(action Action) Action {
	if p.ignoreConflicts {
		action.Kind = ActionSkip
		action.Fatal = false
		return action
	}
	action.Kind = ActionConflict
	action.Fatal = true
	return action
}

// PlanUnlink classifies all selected source files for unlink without mutating the filesystem.
func PlanUnlink(hostRoot string, sources []SourceFiles) (Plan, error) {
	hostRoot = filepath.Clean(hostRoot)
	actions := make([]Action, 0)
	for _, source := range sources {
		for _, relPath := range source.RelPaths {
			action, err := planUnlinkAction(hostRoot, source, relPath)
			if err != nil {
				return Plan{}, err
			}
			actions = append(actions, action)
		}
	}
	return Plan{Actions: actions}, nil
}

func planUnlinkAction(hostRoot string, source SourceFiles, relPath string) (Action, error) {
	action := baseAction(hostRoot, source, relPath)
	info, err := os.Lstat(action.HostPath)
	if err != nil {
		if os.IsNotExist(err) {
			action.Kind = ActionNoop
			action.Reason = "missing"
			return action, nil
		}
		return Action{}, err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		action.Kind = ActionSkip
		if info.IsDir() {
			action.Reason = "host path is a directory"
		} else {
			action.Reason = "host path is not a symlink"
		}
		return action, nil
	}

	ok, err := PointsTo(action.HostPath, action.SourcePath)
	if err != nil {
		return Action{}, err
	}
	if !ok {
		action.Kind = ActionSkip
		action.Reason = "unexpected symlink"
		return action, nil
	}
	action.Kind = ActionRemove
	return action, nil
}

func baseAction(hostRoot string, source SourceFiles, relPath string) Action {
	relPath = filepath.Clean(relPath)
	sourceRoot := filepath.Clean(source.Root)
	return Action{
		SourceName: source.Name,
		SourceRoot: sourceRoot,
		SourcePath: filepath.Join(sourceRoot, relPath),
		HostPath:   filepath.Join(hostRoot, relPath),
		RelPath:    relPath,
	}
}

// ApplyLink applies create actions in plan. It refuses to apply a plan with fatal conflicts.
func ApplyLink(plan Plan) error {
	if plan.HasFatalConflicts() {
		return fmt.Errorf("link plan has fatal conflicts")
	}
	for _, action := range plan.Actions {
		if action.Kind != ActionCreate {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.HostPath), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(action.LinkTarget, action.HostPath); err != nil {
			return err
		}
	}
	return nil
}

// ApplyUnlink removes only remove actions in plan.
func ApplyUnlink(plan Plan) error {
	for _, action := range plan.Actions {
		if action.Kind != ActionRemove {
			continue
		}
		if err := os.Remove(action.HostPath); err != nil {
			return err
		}
	}
	return nil
}

// RenderAction writes a stable, human-readable line for one action.
func RenderAction(w io.Writer, action Action) error {
	prefix := ""
	suffix := sourceSuffix(action.SourceName)
	relPath := filepath.ToSlash(action.RelPath)
	switch action.Kind {
	case ActionCreate:
		prefix = fmt.Sprintf("CREATE %s -> %s", relPath, filepath.ToSlash(action.LinkTarget))
	case ActionRemove:
		prefix = fmt.Sprintf("REMOVE %s", relPath)
	case ActionNoop:
		if action.Reason != "" {
			prefix = fmt.Sprintf("NOOP %s %s", relPath, action.Reason)
		} else {
			prefix = fmt.Sprintf("NOOP %s", relPath)
		}
	case ActionSkip:
		prefix = fmt.Sprintf("SKIP %s %s", relPath, action.Reason)
	case ActionConflict:
		prefix = fmt.Sprintf("CONFLICT %s %s", relPath, action.Reason)
	default:
		prefix = fmt.Sprintf("%s %s %s", action.Kind, relPath, action.Reason)
	}
	_, err := fmt.Fprintf(w, "%s%s\n", prefix, suffix)
	return err
}

// RenderActions writes stable lines for actions.
func RenderActions(w io.Writer, actions []Action) error {
	for _, action := range actions {
		if err := RenderAction(w, action); err != nil {
			return err
		}
	}
	return nil
}

func sourceSuffix(sourceName string) string {
	if sourceName == "" {
		return ""
	}
	return fmt.Sprintf(" [source=%s]", sourceName)
}

// RelativeTarget computes the relative symlink target from hostPath to sourcePath.
func RelativeTarget(hostPath, sourcePath string) (string, error) {
	return filepath.Rel(filepath.Dir(hostPath), sourcePath)
}

// PointsTo reports whether linkPath is a symlink that resolves to targetPath.
func PointsTo(linkPath, targetPath string) (bool, error) {
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	actual, err := os.Readlink(linkPath)
	if err != nil {
		return false, err
	}
	if !filepath.IsAbs(actual) {
		actual = filepath.Join(filepath.Dir(linkPath), actual)
	}
	actual = filepath.Clean(actual)

	expected, err := filepath.Abs(targetPath)
	if err != nil {
		return false, err
	}
	expected = filepath.Clean(expected)

	if resolvedActual, err := filepath.EvalSymlinks(linkPath); err == nil {
		actual = filepath.Clean(resolvedActual)
	}
	if resolvedExpected, err := filepath.EvalSymlinks(targetPath); err == nil {
		expected = filepath.Clean(resolvedExpected)
	}

	return actual == expected, nil
}
