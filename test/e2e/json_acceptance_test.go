package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type jsonActionEnvelope struct {
	DryRun  bool         `json:"dry_run"`
	Actions []jsonAction `json:"actions"`
}

type jsonAction struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Source string `json:"source"`
	Target string `json:"target"`
	Reason string `json:"reason"`
	Fatal  bool   `json:"fatal"`
}

type jsonStatusEnvelope struct {
	Links []jsonStatusLink `json:"links"`
}

type jsonStatusLink struct {
	State   string   `json:"state"`
	Path    string   `json:"path"`
	Source  string   `json:"source"`
	Profile string   `json:"profile"`
	Target  string   `json:"target"`
	Reasons []string `json:"reasons"`
}

type jsonDoctorEnvelope struct {
	Healthy bool              `json:"healthy"`
	Issues  []jsonDoctorIssue `json:"issues"`
}

type jsonDoctorIssue struct {
	Kind    string   `json:"kind"`
	Source  string   `json:"source"`
	Message string   `json:"message"`
	Pattern string   `json:"pattern"`
	Profile string   `json:"profile"`
	Path    string   `json:"path"`
	Target  string   `json:"target"`
	Reasons []string `json:"reasons"`
	Reason  string   `json:"reason"`
}

type jsonPruneEnvelope struct {
	Removed []jsonPruneRemoved `json:"removed"`
}

type jsonPruneRemoved struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func TestInventoryJSONAcceptance(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"client\", \"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	profiles := runCubby(t, bin, host, "profile", "list", "--json")
	if profiles.code != 0 {
		t.Fatalf("profile list --json code = %d, stderr = %s", profiles.code, profiles.stderr)
	}
	if want := "{\"profiles\":[\"client\",\"personal\",\"work\"]}\n"; profiles.stdout != want {
		t.Fatalf("profile list --json stdout = %q, want %q", profiles.stdout, want)
	}

	sources := runCubby(t, bin, host, "source", "list", "--json")
	if sources.code != 0 {
		t.Fatalf("source list --json code = %d, stderr = %s", sources.code, sources.stderr)
	}
	wantSources := "{\"sources\":[{\"name\":\"one\",\"path\":\"" + filepath.ToSlash(filepath.Clean(src1)) + "\",\"profiles\":[\"work\",\"personal\"]},{\"name\":\"two\",\"path\":\"" + filepath.ToSlash(filepath.Clean(src2)) + "\",\"profiles\":[\"client\",\"work\"]}]}\n"
	if sources.stdout != wantSources {
		t.Fatalf("source list --json stdout = %q, want %q", sources.stdout, wantSources)
	}
}

func TestGitignoreJSONAcceptance(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")

	check := runCubby(t, bin, host, "gitignore", "check", "--json")
	if check.code == 0 {
		t.Fatalf("gitignore check --json code = 0, want missing-pattern failure")
	}
	var checkBody struct {
		OK      bool     `json:"ok"`
		Missing []string `json:"missing"`
	}
	decodeJSON(t, check.stdout, &checkBody)
	if checkBody.OK || !reflect.DeepEqual(checkBody.Missing, []string{"/.cubby.toml", "*.work.*", "*.work"}) {
		t.Fatalf("gitignore check JSON = %#v, want missing host config and work patterns", checkBody)
	}

	sync := runCubby(t, bin, host, "gitignore", "sync", "--json")
	if sync.code != 0 {
		t.Fatalf("gitignore sync --json code = %d, stderr = %s", sync.code, sync.stderr)
	}
	var syncBody struct {
		Changed bool     `json:"changed"`
		Added   []string `json:"added"`
	}
	decodeJSON(t, sync.stdout, &syncBody)
	if !syncBody.Changed || !reflect.DeepEqual(syncBody.Added, []string{"/.cubby.toml", "*.work.*", "*.work"}) {
		t.Fatalf("gitignore sync JSON = %#v, want changed host config and work patterns", syncBody)
	}

	syncAgain := runCubby(t, bin, host, "gitignore", "sync", "--json")
	if syncAgain.code != 0 {
		t.Fatalf("second gitignore sync --json code = %d, stderr = %s", syncAgain.code, syncAgain.stderr)
	}
	var syncAgainBody struct {
		Changed bool     `json:"changed"`
		Added   []string `json:"added"`
	}
	decodeJSON(t, syncAgain.stdout, &syncAgainBody)
	if syncAgainBody.Changed || len(syncAgainBody.Added) != 0 {
		t.Fatalf("second gitignore sync JSON = %#v, want unchanged empty added", syncAgainBody)
	}
}

func TestLinkJSONDryRunAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "create.work"), "create\n")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	result := runCubby(t, bin, host, "link", "--json", "--dry-run")
	if result.code == 0 {
		t.Fatalf("link --json --dry-run code = 0, want conflict-equivalent failure")
	}
	var body jsonActionEnvelope
	decodeJSON(t, result.stdout, &body)
	if !body.DryRun {
		t.Fatalf("dry_run = false, want true")
	}
	create := findJSONAction(t, body.Actions, "create", "create.work")
	if create.Source != "src" || create.Target == "" {
		t.Fatalf("create action = %#v, want source and target", create)
	}
	noop := findJSONAction(t, body.Actions, "noop", "linked.work")
	if noop.Reason != "already linked" {
		t.Fatalf("noop action = %#v, want already linked reason", noop)
	}
	conflict := findJSONAction(t, body.Actions, "conflict", "conflict.work")
	if !conflict.Fatal || conflict.Reason != "host path already exists" {
		t.Fatalf("conflict action = %#v, want fatal host path conflict", conflict)
	}
	assertNotExist(t, filepath.Join(host, "create.work"))
	if got := mustRead(t, filepath.Join(host, "conflict.work")); got != "host conflict\n" {
		t.Fatalf("conflict file = %q, want untouched", got)
	}
}

func TestLinkJSONFatalConflictAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "create.work"), "create\n")
	mustWrite(t, filepath.Join(src, "conflict.work"), "source conflict\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(host, "conflict.work"), "host conflict\n")

	result := runCubby(t, bin, host, "link", "--json")
	if result.code == 0 {
		t.Fatalf("link --json code = 0, want conflict failure")
	}
	var body jsonActionEnvelope
	decodeJSON(t, result.stdout, &body)
	if body.DryRun {
		t.Fatalf("dry_run = true, want false")
	}
	findJSONAction(t, body.Actions, "create", "create.work")
	conflict := findJSONAction(t, body.Actions, "conflict", "conflict.work")
	if !conflict.Fatal {
		t.Fatalf("conflict action = %#v, want fatal", conflict)
	}
	assertNotExist(t, filepath.Join(host, "create.work"))
	if got := mustRead(t, filepath.Join(host, "conflict.work")); got != "host conflict\n" {
		t.Fatalf("conflict file = %q, want untouched", got)
	}
}

func TestUnlinkJSONDryRunAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	other := filepath.Join(tmp, "other")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "linked.work"), "linked\n")
	mustWrite(t, filepath.Join(src, "regular.work"), "regular source\n")
	mustWrite(t, filepath.Join(src, "unexpected.work"), "unexpected source\n")
	mustWrite(t, filepath.Join(src, "missing.work"), "missing source\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "linked.work"), filepath.Join(src, "linked.work"))
	mustWrite(t, filepath.Join(host, "regular.work"), "host regular\n")
	mustWrite(t, filepath.Join(other, "unexpected.work"), "other\n")
	mustSymlink(t, filepath.Join(host, "unexpected.work"), filepath.Join(other, "unexpected.work"))

	result := runCubby(t, bin, host, "unlink", "--json", "--dry-run")
	if result.code != 0 {
		t.Fatalf("unlink --json --dry-run code = %d, stderr = %s", result.code, result.stderr)
	}
	var body jsonActionEnvelope
	decodeJSON(t, result.stdout, &body)
	if !body.DryRun {
		t.Fatalf("dry_run = false, want true")
	}
	findJSONAction(t, body.Actions, "remove", "linked.work")
	if regular := findJSONAction(t, body.Actions, "skip", "regular.work"); regular.Reason == "" {
		t.Fatalf("regular skip = %#v, want reason", regular)
	}
	if unexpected := findJSONAction(t, body.Actions, "skip", "unexpected.work"); unexpected.Reason != "unexpected symlink" {
		t.Fatalf("unexpected skip = %#v, want unexpected symlink", unexpected)
	}
	if missing := findJSONAction(t, body.Actions, "noop", "missing.work"); missing.Reason != "missing" {
		t.Fatalf("missing noop = %#v, want missing", missing)
	}
	assertSymlinkExists(t, filepath.Join(host, "linked.work"))
	if got := mustRead(t, filepath.Join(host, "regular.work")); got != "host regular\n" {
		t.Fatalf("regular file = %q, want untouched", got)
	}
}

func TestStatusJSONAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "ok.work"), "ok\n")
	mustWrite(t, filepath.Join(src, "other.work"), "other\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "ok.work"), filepath.Join(src, "ok.work"))
	mustSymlink(t, filepath.Join(host, "bad.work"), filepath.Join(src, "other.work"))

	result := runCubby(t, bin, host, "status", "--json")
	if result.code != 0 {
		t.Fatalf("status --json code = %d, stderr = %s", result.code, result.stderr)
	}
	var body jsonStatusEnvelope
	decodeJSON(t, result.stdout, &body)
	linked := findStatusLink(t, body.Links, "ok.work")
	if linked.State != "linked" || linked.Source != "src" || linked.Profile != "work" || linked.Target != "ok.work" || len(linked.Reasons) != 0 {
		t.Fatalf("linked status = %#v", linked)
	}
	drift := findStatusLink(t, body.Links, "bad.work")
	if drift.State != "drift" || drift.Target != "other.work" || !stringSliceContains(drift.Reasons, "path mismatch") {
		t.Fatalf("drift status = %#v, want path mismatch", drift)
	}
}

func TestDoctorJSONAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "ok.work"), "ok\n")

	healthyHost := filepath.Join(tmp, "healthy")
	mustWrite(t, filepath.Join(healthyHost, ".cubby.toml"), "profiles = [\"work\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustWrite(t, filepath.Join(healthyHost, ".gitignore"), "/.cubby.toml\n*.work.*\n*.work\n")
	mustSymlink(t, filepath.Join(healthyHost, "ok.work"), filepath.Join(src, "ok.work"))
	healthy := runCubby(t, bin, healthyHost, "doctor", "--json")
	if healthy.code != 0 {
		t.Fatalf("healthy doctor --json code = %d, stderr = %s", healthy.code, healthy.stderr)
	}
	var healthyBody jsonDoctorEnvelope
	decodeJSON(t, healthy.stdout, &healthyBody)
	if !healthyBody.Healthy || len(healthyBody.Issues) != 0 {
		t.Fatalf("healthy doctor JSON = %#v", healthyBody)
	}

	unhealthyHost := filepath.Join(tmp, "unhealthy")
	mustWrite(t, filepath.Join(unhealthyHost, ".cubby.toml"), "profiles = [\"work\", \"ghost\"]\n\n[[source]]\nname = \"src\"\npath = \""+src+"\"\n\n[[source]]\nname = \"missing\"\npath = \""+filepath.Join(tmp, "missing")+"\"\n")
	mustWrite(t, filepath.Join(src, "stale.work"), "stale\n")
	if err := os.Remove(filepath.Join(src, "stale.work")); err != nil {
		t.Fatalf("Remove stale source error = %v", err)
	}
	mustWrite(t, filepath.Join(unhealthyHost, "ok.work"), "conflict\n")
	mustSymlink(t, filepath.Join(unhealthyHost, "stale.work"), filepath.Join(src, "stale.work"))
	unhealthy := runCubby(t, bin, unhealthyHost, "doctor", "--json")
	if unhealthy.code == 0 {
		t.Fatalf("unhealthy doctor --json code = 0, want failure")
	}
	var unhealthyBody jsonDoctorEnvelope
	decodeJSON(t, unhealthy.stdout, &unhealthyBody)
	if unhealthyBody.Healthy {
		t.Fatalf("unhealthy doctor healthy = true")
	}
	for _, kind := range []string{"missing_source", "missing_gitignore", "missing_profile", "dangling", "conflict"} {
		if !doctorIssueKindPresent(unhealthyBody.Issues, kind) {
			t.Fatalf("doctor JSON issues missing kind %q: %#v", kind, unhealthyBody.Issues)
		}
	}
}

func TestPruneJSONAcceptance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src := filepath.Join(tmp, "src")
	mustWrite(t, filepath.Join(src, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(src, "valid.work"), "valid\n")
	mustWrite(t, filepath.Join(src, "stale.work"), "stale\n")
	if err := os.Remove(filepath.Join(src, "stale.work")); err != nil {
		t.Fatalf("Remove stale source error = %v", err)
	}
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"src\"\npath = \""+src+"\"\n")
	mustSymlink(t, filepath.Join(host, "stale.work"), filepath.Join(src, "stale.work"))
	mustSymlink(t, filepath.Join(host, "valid.work"), filepath.Join(src, "valid.work"))

	result := runCubby(t, bin, host, "prune", "--json")
	if result.code != 0 {
		t.Fatalf("prune --json code = %d, stderr = %s", result.code, result.stderr)
	}
	var body jsonPruneEnvelope
	decodeJSON(t, result.stdout, &body)
	if len(body.Removed) != 1 || body.Removed[0].Path != "stale.work" || body.Removed[0].Source != "src" || body.Removed[0].Target != "stale.work" {
		t.Fatalf("prune JSON = %#v, want stale removal", body)
	}
	assertNotExist(t, filepath.Join(host, "stale.work"))
	assertSymlinkExists(t, filepath.Join(host, "valid.work"))

	again := runCubby(t, bin, host, "prune", "--json")
	if again.code != 0 {
		t.Fatalf("second prune --json code = %d, stderr = %s", again.code, again.stderr)
	}
	var againBody jsonPruneEnvelope
	decodeJSON(t, again.stdout, &againBody)
	if len(againBody.Removed) != 0 {
		t.Fatalf("second prune JSON = %#v, want empty removed", againBody)
	}
}

func decodeJSON(t *testing.T, text string, value any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		t.Fatalf("decode JSON %q error = %v", text, err)
	}
	if decoder.More() {
		t.Fatalf("JSON output has trailing values: %q", text)
	}
}

func findJSONAction(t *testing.T, actions []jsonAction, kind, path string) jsonAction {
	t.Helper()
	for _, action := range actions {
		if action.Kind == kind && action.Path == path {
			return action
		}
	}
	t.Fatalf("missing action kind=%s path=%s in %#v", kind, path, actions)
	return jsonAction{}
}

func findStatusLink(t *testing.T, links []jsonStatusLink, path string) jsonStatusLink {
	t.Helper()
	for _, link := range links {
		if link.Path == path {
			return link
		}
	}
	t.Fatalf("missing status link path=%s in %#v", path, links)
	return jsonStatusLink{}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func doctorIssueKindPresent(issues []jsonDoctorIssue, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}
