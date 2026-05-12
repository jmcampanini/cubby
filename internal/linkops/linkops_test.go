package linkops

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRelativeTargetForNestedHostPath(t *testing.T) {
	root := t.TempDir()
	hostPath := filepath.Join(root, "host", "nvim", "init.work.lua")
	sourcePath := filepath.Join(root, "src", "nvim", "init.work.lua")

	got, err := RelativeTarget(hostPath, sourcePath)
	if err != nil {
		t.Fatalf("RelativeTarget() error = %v", err)
	}
	want := filepath.Join("..", "..", "src", "nvim", "init.work.lua")
	if got != want {
		t.Fatalf("RelativeTarget() = %q, want %q", got, want)
	}
}

func TestPointsToDetectsCorrectRelativeSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "src", "nvim", "init.work.lua")
	link := filepath.Join(root, "host", "nvim", "init.work.lua")
	mustWrite(t, target, "-- work\n")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(link), err)
	}
	rel, err := RelativeTarget(link, target)
	if err != nil {
		t.Fatalf("RelativeTarget() error = %v", err)
	}
	if err := os.Symlink(rel, link); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", rel, link, err)
	}

	ok, err := PointsTo(link, target)
	if err != nil {
		t.Fatalf("PointsTo() error = %v", err)
	}
	if !ok {
		t.Fatalf("PointsTo() = false, want true")
	}
}

func TestPlanLinkClassifiesHostPathStates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	mustWrite(t, filepath.Join(src, "missing.work"), "missing\n")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "regular.work"), "regular source\n")
	mustWrite(t, filepath.Join(src, "dir.work"), "dir source\n")
	mustWrite(t, filepath.Join(src, "unexpected.work"), "unexpected source\n")
	mustWrite(t, filepath.Join(host, "regular.work"), "regular host\n")
	mustMkdir(t, filepath.Join(host, "dir.work"))
	mustWrite(t, filepath.Join(other, "unexpected.work"), "other\n")
	mustSymlinkTo(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))
	mustSymlinkTo(t, filepath.Join(host, "unexpected.work"), filepath.Join(other, "unexpected.work"))

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"missing.work", "linked.work", "regular.work", "dir.work", "unexpected.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "missing.work", ActionCreate, false, "")
	assertAction(t, plan, "linked.work", ActionNoop, false, "already linked")
	assertAction(t, plan, "regular.work", ActionConflict, true, "already exists")
	assertAction(t, plan, "dir.work", ActionConflict, true, "directory")
	assertAction(t, plan, "unexpected.work", ActionConflict, true, "unexpected symlink")
}

func TestPlanLinkIgnoreConflictsConvertsConflictsToSkips(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "file.work"), "source\n")
	mustWrite(t, filepath.Join(host, "file.work"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"file.work"}}}, PlanOptions{IgnoreConflicts: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "file.work", ActionSkip, false, "already exists")
	if plan.HasFatalConflicts() {
		t.Fatalf("HasFatalConflicts() = true, want false")
	}
}

func TestPlanLinkClassifiesCrossSourceCollisions(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src1 := filepath.Join(root, "src1")
	src2 := filepath.Join(root, "src2")
	mustWrite(t, filepath.Join(src1, "same.work"), "one\n")
	mustWrite(t, filepath.Join(src2, "same.work"), "two\n")

	plan, err := PlanLink(host, []SourceFiles{
		{Name: "one", Root: src1, RelPaths: []string{"same.work"}},
		{Name: "two", Root: src2, RelPaths: []string{"same.work"}},
	}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionCreate || plan.Actions[0].SourceName != "one" {
		t.Fatalf("winner action = %+v, want create from one", plan.Actions[0])
	}
	if plan.Actions[1].Kind != ActionConflict || !plan.Actions[1].Fatal || !strings.Contains(plan.Actions[1].Reason, "collision") {
		t.Fatalf("collision action = %+v, want fatal collision", plan.Actions[1])
	}

	ignored, err := PlanLink(host, []SourceFiles{
		{Name: "one", Root: src1, RelPaths: []string{"same.work"}},
		{Name: "two", Root: src2, RelPaths: []string{"same.work"}},
	}, PlanOptions{IgnoreConflicts: true})
	if err != nil {
		t.Fatalf("PlanLink(ignore) error = %v", err)
	}
	if ignored.Actions[1].Kind != ActionSkip || ignored.Actions[1].Fatal {
		t.Fatalf("ignored collision action = %+v, want nonfatal skip", ignored.Actions[1])
	}
}

func TestApplyLinkRefusesFatalConflictPlanBeforeCreates(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(src, "create.work"), "source create\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"conflict.work", "create.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if err := ApplyLink(plan); err == nil {
		t.Fatalf("ApplyLink() error = nil, want fatal conflict refusal")
	}
	if _, err := os.Lstat(filepath.Join(host, "create.work")); !os.IsNotExist(err) {
		t.Fatalf("create.work Lstat error = %v, want not exist", err)
	}
}

func TestPlanUnlinkClassifiesRemoveNoopAndSkips(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "regular.work"), "regular source\n")
	mustWrite(t, filepath.Join(src, "unexpected.work"), "unexpected source\n")
	mustWrite(t, filepath.Join(src, "missing.work"), "missing source\n")
	mustWrite(t, filepath.Join(host, "regular.work"), "regular host\n")
	mustWrite(t, filepath.Join(other, "unexpected.work"), "other\n")
	mustSymlinkTo(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))
	mustSymlinkTo(t, filepath.Join(host, "unexpected.work"), filepath.Join(other, "unexpected.work"))

	plan, err := PlanUnlink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"linked.work", "regular.work", "unexpected.work", "missing.work"}}})
	if err != nil {
		t.Fatalf("PlanUnlink() error = %v", err)
	}
	assertAction(t, plan, "linked.work", ActionRemove, false, "")
	assertAction(t, plan, "regular.work", ActionSkip, false, "not a symlink")
	assertAction(t, plan, "unexpected.work", ActionSkip, false, "unexpected symlink")
	assertAction(t, plan, "missing.work", ActionNoop, false, "missing")
}

func TestRenderActionUsesStablePrefixes(t *testing.T) {
	var buf bytes.Buffer
	action := Action{Kind: ActionSkip, RelPath: filepath.Join("nvim", "init.work.lua"), Reason: "host path already exists", SourceName: "src"}
	if err := RenderAction(&buf, action); err != nil {
		t.Fatalf("RenderAction() error = %v", err)
	}
	wantPrefix := "SKIP nvim/init.work.lua host path already exists [source=src]\n"
	if buf.String() != wantPrefix {
		t.Fatalf("RenderAction() = %q, want %q", buf.String(), wantPrefix)
	}
}

func assertAction(t *testing.T, plan Plan, relPath string, kind ActionKind, fatal bool, reasonContains string) {
	t.Helper()
	for _, action := range plan.Actions {
		if action.RelPath != filepath.Clean(relPath) {
			continue
		}
		if action.Kind != kind || action.Fatal != fatal || (reasonContains != "" && !strings.Contains(action.Reason, reasonContains)) {
			t.Fatalf("action for %q = %+v, want kind=%s fatal=%v reason containing %q", relPath, action, kind, fatal, reasonContains)
		}
		return
	}
	t.Fatalf("action for %q not found in %+v", relPath, plan.Actions)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustSymlinkTo(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(linkPath), err)
	}
	target, err := RelativeTarget(linkPath, targetPath)
	if err != nil {
		t.Fatalf("RelativeTarget() error = %v", err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", target, linkPath, err)
	}
}
