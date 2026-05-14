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

func TestPlanLinkCaseInsensitiveCollisionWithinSameSource(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"foo.work", "FOO.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionCreate {
		t.Fatalf("first action = %+v, want create", plan.Actions[0])
	}
	if plan.Actions[1].Kind != ActionConflict || !plan.Actions[1].Fatal {
		t.Fatalf("second action = %+v, want fatal conflict", plan.Actions[1])
	}
	if !strings.Contains(plan.Actions[1].Reason, "path case collision") || !strings.Contains(plan.Actions[1].Reason, "foo.work") {
		t.Fatalf("case collision reason = %q, want path case collision with winner", plan.Actions[1].Reason)
	}
}

func TestPlanLinkCaseInsensitiveCollisionAcrossSources(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src1 := filepath.Join(root, "src1")
	src2 := filepath.Join(root, "src2")

	plan, err := PlanLink(host, []SourceFiles{
		{Name: "one", Root: src1, RelPaths: []string{"foo.work"}},
		{Name: "two", Root: src2, RelPaths: []string{"FOO.work"}},
	}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if plan.Actions[0].Kind != ActionCreate || plan.Actions[0].RelPath != "foo.work" {
		t.Fatalf("winner action = %+v, want foo.work create", plan.Actions[0])
	}
	if plan.Actions[1].Kind != ActionConflict || !plan.Actions[1].Fatal || !strings.Contains(plan.Actions[1].Reason, "path case collision") {
		t.Fatalf("case collision action = %+v, want fatal path case collision", plan.Actions[1])
	}
}

func TestPlanLinkCaseInsensitiveCollisionCanBeSkipped(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"foo.work", "FOO.work", "bar.work"}}}, PlanOptions{IgnoreConflicts: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "foo.work", ActionCreate, false, "")
	assertAction(t, plan, "FOO.work", ActionSkip, false, "path case collision")
	assertAction(t, plan, "bar.work", ActionCreate, false, "")
}

func TestApplyLinkCreatesOnlyNonSkippedCaseCollisionActions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(src, "foo.work"), "foo\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"foo.work", "FOO.work", "bar.work"}}}, PlanOptions{IgnoreConflicts: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if err := ApplyLink(plan); err != nil {
		t.Fatalf("ApplyLink() error = %v", err)
	}
	assertSymlinkExists(t, filepath.Join(host, "foo.work"))
	assertSymlinkExists(t, filepath.Join(host, "bar.work"))
}

func TestPlanLinkHostCaseConflictIsFatalByDefault(t *testing.T) {
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"FOO.work", "bar.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "FOO.work", ActionConflict, true, "host path case conflict with foo.work")
	assertAction(t, plan, "bar.work", ActionCreate, false, "")
}

func TestPlanLinkHostCaseConflictIncludesParentPathVariants(t *testing.T) {
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "Nvim", "init.work.lua"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{filepath.Join("nvim", "init.work.lua")}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, filepath.Join("nvim", "init.work.lua"), ActionConflict, true, "host path case conflict with Nvim")
}

func TestPlanLinkHostCaseConflictFindsLeafVariantUnderExactParent(t *testing.T) {
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "nvim", "Init.work.lua"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{filepath.Join("nvim", "init.work.lua")}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, filepath.Join("nvim", "init.work.lua"), ActionConflict, true, "host path case conflict with nvim/Init.work.lua")
}

func TestPlanLinkHostCaseConflictSearchesSiblingParentVariants(t *testing.T) {
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustMkdir(t, filepath.Join(host, "foo"))
	mustWrite(t, filepath.Join(host, "Foo", "bar.work"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{filepath.Join("foo", "bar.work")}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, filepath.Join("foo", "bar.work"), ActionConflict, true, "host path case conflict with Foo")
}

func TestPlanLinkHostCaseConflictCanBeSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"FOO.work", "bar.work"}}}, PlanOptions{IgnoreConflicts: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "FOO.work", ActionSkip, false, "host path case conflict with foo.work")
	assertAction(t, plan, "bar.work", ActionCreate, false, "")
	if err := ApplyLink(plan); err != nil {
		t.Fatalf("ApplyLink() error = %v", err)
	}
	assertSymlinkExists(t, filepath.Join(host, "bar.work"))
}

func TestPlanLinkHostCaseConflictPreventsApplyCreates(t *testing.T) {
	root := t.TempDir()
	requireCaseSensitiveFilesystem(t, root)
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")
	mustWrite(t, filepath.Join(src, "bar.work"), "bar\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"FOO.work", "bar.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if err := ApplyLink(plan); err == nil {
		t.Fatalf("ApplyLink() error = nil, want fatal conflict refusal")
	}
	if _, err := os.Lstat(filepath.Join(host, "bar.work")); !os.IsNotExist(err) {
		t.Fatalf("bar.work Lstat error = %v, want not exist", err)
	}
}

func TestPlanLinkCaseSensitiveDoesNotUseHostCasePolicyConflict(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "foo.work"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"FOO.work"}}}, PlanOptions{CaseSensitive: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(plan.Actions))
	}
	if strings.Contains(plan.Actions[0].Reason, "host path case conflict") {
		t.Fatalf("case-sensitive action = %+v, want no Cubby host case-policy conflict", plan.Actions[0])
	}
}

func TestPlanLinkExactPathDoesNotRequireReadableParentDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions differ on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	parent := filepath.Join(host, "locked")
	mustWrite(t, filepath.Join(parent, "file.work"), "host\n")
	if err := os.Chmod(parent, 0o111); err != nil {
		t.Fatalf("Chmod(%q) error = %v", parent, err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0o755); err != nil {
			t.Fatalf("restore Chmod(%q) error = %v", parent, err)
		}
	})

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{filepath.Join("locked", "file.work")}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v, want exact-path classification without reading parent", err)
	}
	assertAction(t, plan, filepath.Join("locked", "file.work"), ActionConflict, true, "host path already exists")
}

func TestPlanLinkExactPathStillUsesExactClassification(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWrite(t, filepath.Join(host, "FOO.work"), "host\n")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"FOO.work"}}}, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "FOO.work", ActionConflict, true, "host path already exists")
}

func TestPlanLinkCaseSensitiveAllowsCaseDistinctPaths(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")

	plan, err := PlanLink(host, []SourceFiles{{Name: "src", Root: src, RelPaths: []string{"foo.work", "FOO.work"}}}, PlanOptions{CaseSensitive: true})
	if err != nil {
		t.Fatalf("PlanLink() error = %v", err)
	}
	assertAction(t, plan, "foo.work", ActionCreate, false, "")
	assertAction(t, plan, "FOO.work", ActionCreate, false, "")
}

func TestRenderCaseCollisionIncludesWinnerRelpath(t *testing.T) {
	var buf bytes.Buffer
	action := Action{Kind: ActionConflict, RelPath: "FOO.work", Reason: "path case collision with foo.work", SourceName: "src", Fatal: true}
	if err := RenderAction(&buf, action); err != nil {
		t.Fatalf("RenderAction() error = %v", err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "CONFLICT FOO.work") || !strings.Contains(got, "path case collision with foo.work") {
		t.Fatalf("RenderAction() = %q, want conflict with case collision winner", got)
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

func requireCaseSensitiveFilesystem(t *testing.T, dir string) {
	t.Helper()
	probe := filepath.Join(dir, "case-probe")
	mustWrite(t, probe, "probe\n")
	if _, err := os.Lstat(filepath.Join(dir, "CASE-PROBE")); err == nil {
		t.Skip("filesystem is case-insensitive; exact missing-path case-variant behavior is not observable")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Lstat case-sensitivity probe error = %v", err)
	}
}

func assertSymlinkExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
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
