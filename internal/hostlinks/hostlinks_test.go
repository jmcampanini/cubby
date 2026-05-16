package hostlinks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jmcampanini/cubby/internal/config"
)

func TestDiscoverClassifiesManagedSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	mustWriteHostlinks(t, filepath.Join(src, "file.work"), "work\n")
	mustWriteHostlinks(t, filepath.Join(src, "abs.work"), "work\n")
	mustWriteHostlinks(t, filepath.Join(other, "file.work"), "other\n")
	sources := []config.RegisteredSource{testSource("src", src, []string{"work"}, nil)}

	mustSymlinkHostlinks(t, filepath.Join(host, "file.work"), filepath.Join(src, "file.work"))
	if err := os.MkdirAll(host, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src, "abs.work"), filepath.Join(host, "abs.work")); err != nil {
		t.Fatalf("absolute Symlink error = %v", err)
	}
	mustSymlinkHostlinks(t, filepath.Join(host, "outside.work"), filepath.Join(other, "file.work"))

	links, err := Discover(host, sources)
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("links len = %d, want 2: %#v", len(links), links)
	}
	for _, link := range links {
		if link.SourceName != "src" || link.Profile != "work" || !link.TargetExists {
			t.Fatalf("link = %#v, want managed src/work existing", link)
		}
	}
}

func TestDiscoverDetectsDanglingDriftProfileAndIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	mustWriteHostlinks(t, filepath.Join(src, "file.work"), "work\n")
	mustWriteHostlinks(t, filepath.Join(src, "ignored.work"), "work\n")
	mustWriteHostlinks(t, filepath.Join(src, "plain.txt"), "plain\n")
	sources := []config.RegisteredSource{testSource("src", src, []string{"work"}, []string{"ignored.work"})}

	mustSymlinkHostlinks(t, filepath.Join(host, "elsewhere.work"), filepath.Join(src, "file.work"))
	mustSymlinkHostlinks(t, filepath.Join(host, "stale.work"), filepath.Join(src, "stale.work"))
	mustSymlinkHostlinks(t, filepath.Join(host, "plain.txt"), filepath.Join(src, "plain.txt"))
	mustSymlinkHostlinks(t, filepath.Join(host, "ignored.work"), filepath.Join(src, "ignored.work"))

	links, err := Discover(host, sources)
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	byRel := map[string]ManagedLink{}
	for _, link := range links {
		byRel[link.HostRelPath] = link
	}
	assertReason(t, byRel["elsewhere.work"], ReasonPathMismatch)
	assertReason(t, byRel["stale.work"], ReasonDangling)
	assertReason(t, byRel["plain.txt"], ReasonUnknown)
	assertReason(t, byRel["ignored.work"], ReasonIgnored)
}

func TestDiscoverDetectsUnresolvedAndNonRegularTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", src, err)
	}
	if err := os.Symlink("loop.work", filepath.Join(src, "loop.work")); err != nil {
		t.Fatalf("Symlink(loop.work) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "dir.work"), 0o755); err != nil {
		t.Fatalf("MkdirAll(dir.work) error = %v", err)
	}
	sources := []config.RegisteredSource{testSource("src", src, []string{"work"}, nil)}
	mustSymlinkHostlinks(t, filepath.Join(host, "loop.work"), filepath.Join(src, "loop.work"))
	mustSymlinkHostlinks(t, filepath.Join(host, "dir.work"), filepath.Join(src, "dir.work"))

	links, err := Discover(host, sources)
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	byRel := map[string]ManagedLink{}
	for _, link := range links {
		byRel[link.HostRelPath] = link
	}
	assertReason(t, byRel["loop.work"], ReasonUnresolved)
	assertReason(t, byRel["dir.work"], ReasonNonRegular)
}

func TestDiscoverClassifiesDanglingLinksThroughResolvedSourceAlias(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	realSrc := filepath.Join(root, "real-src")
	aliasSrc := filepath.Join(root, "alias-src")
	mustWriteHostlinks(t, filepath.Join(realSrc, "existing.work"), "work\n")
	if err := os.Symlink(realSrc, aliasSrc); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", realSrc, aliasSrc, err)
	}
	resolvedRealSrc, err := filepath.EvalSymlinks(realSrc)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", realSrc, err)
	}
	mustSymlinkHostlinks(t, filepath.Join(host, "stale.work"), filepath.Join(resolvedRealSrc, "stale.work"))
	sources := []config.RegisteredSource{testSource("src", aliasSrc, []string{"work"}, nil)}

	links, err := Discover(host, sources)
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("links len = %d, want 1: %#v", len(links), links)
	}
	if links[0].SourceName != "src" || links[0].SourceRelPath != "stale.work" || links[0].TargetExists {
		t.Fatalf("link = %#v, want dangling link under source alias", links[0])
	}
	assertReason(t, links[0], ReasonDangling)
}

func TestDiscoverDiagnosticsClassifiesDanglingLinksUnderMissingSourceRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	missingSrc := filepath.Join(root, "missing-src")
	other := filepath.Join(root, "other")
	mustWriteHostlinks(t, filepath.Join(other, "placeholder"), "other\n")
	managed := filepath.Join(host, "stale.work")
	unmanaged := filepath.Join(host, "outside.work")
	mustSymlinkHostlinks(t, managed, filepath.Join(missingSrc, "stale.work"))
	mustSymlinkHostlinks(t, unmanaged, filepath.Join(other, "gone.work"))

	links, err := DiscoverDiagnostics(host, nil, []config.DiagnosticSourceRoot{{Name: "missing", ResolvedPath: missingSrc, Order: 0}})
	if err != nil {
		t.Fatalf("DiscoverDiagnostics error = %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("links len = %d, want 1: %#v", len(links), links)
	}
	if links[0].HostRelPath != "stale.work" || links[0].SourceName != "missing" || links[0].TargetExists {
		t.Fatalf("link = %#v, want dangling managed link under missing source", links[0])
	}
	assertReason(t, links[0], ReasonDangling)
}

func TestDiscoverSkipsRegisteredSourceSymlinkRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	realSrc := filepath.Join(root, "real-src")
	hostSourceLink := filepath.Join(host, "src")
	mustWriteHostlinks(t, filepath.Join(realSrc, "file.work"), "work\n")
	if err := os.MkdirAll(host, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", host, err)
	}
	if err := os.Symlink(realSrc, hostSourceLink); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", realSrc, hostSourceLink, err)
	}

	links, err := Discover(host, []config.RegisteredSource{testSource("src", hostSourceLink, []string{"work"}, nil)})
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("links = %#v, want registered source symlink root skipped", links)
	}
}

func TestDiscoverPrefersMostSpecificSourceRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	host := filepath.Join(root, "host")
	src := filepath.Join(root, "src")
	nested := filepath.Join(src, "nested")
	mustWriteHostlinks(t, filepath.Join(nested, "file.work"), "work\n")
	sources := []config.RegisteredSource{
		testSource("outer", src, []string{"work"}, nil),
		testSource("inner", nested, []string{"work"}, nil),
	}
	mustSymlinkHostlinks(t, filepath.Join(host, "file.work"), filepath.Join(nested, "file.work"))

	links, err := Discover(host, sources)
	if err != nil {
		t.Fatalf("Discover error = %v", err)
	}
	if len(links) != 1 || links[0].SourceName != "inner" {
		t.Fatalf("links = %#v, want inner owner", links)
	}
}

func testSource(name, root string, profiles, ignore []string) config.RegisteredSource {
	return config.RegisteredSource{
		HostSource:   config.HostSource{Name: name, Path: root},
		ResolvedPath: filepath.Clean(root),
		Config:       config.SourceConfig{Profiles: profiles, Ignore: ignore},
	}
}

func assertReason(t *testing.T, link ManagedLink, reason string) {
	t.Helper()
	for _, got := range link.DriftReasons {
		if got == reason {
			return
		}
	}
	t.Fatalf("link %#v missing reason %q", link, reason)
}

func mustWriteHostlinks(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mustSymlinkHostlinks(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(linkPath), err)
	}
	target, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", target, linkPath, err)
	}
}
