package e2e_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type operandFixture struct {
	host   string
	source string
}

type treeSnapshotEntry struct {
	path    string
	kind    string
	perm    fs.FileMode
	content []byte
	target  string
}

func TestMutatingCommandsRejectExtraOperand(t *testing.T) {
	bin := buildCubby(t)
	tests := []struct {
		name           string
		args           []string
		commandPath    string
		needsSymlinks  bool
		prepareFixture func(*testing.T) operandFixture
	}{
		{
			name:           "link",
			args:           []string{"link", "unexpected"},
			commandPath:    "cubby link",
			needsSymlinks:  true,
			prepareFixture: prepareLinkOperandFixture,
		},
		{
			name:           "unlink",
			args:           []string{"unlink", "unexpected"},
			commandPath:    "cubby unlink",
			needsSymlinks:  true,
			prepareFixture: prepareUnlinkOperandFixture,
		},
		{
			name:           "prune",
			args:           []string{"prune", "unexpected"},
			commandPath:    "cubby prune",
			needsSymlinks:  true,
			prepareFixture: preparePruneOperandFixture,
		},
		{
			name:           "gitignore_sync",
			args:           []string{"gitignore", "sync", "unexpected"},
			commandPath:    "cubby gitignore sync",
			prepareFixture: prepareGitignoreSyncOperandFixture,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && tt.needsSymlinks {
				t.Skip("symlink privileges vary on Windows")
			}
			fixture := tt.prepareFixture(t)
			hostBefore := snapshotOperandTree(t, fixture.host)
			sourceBefore := snapshotOperandTree(t, fixture.source)

			result := runCubby(t, bin, fixture.host, tt.args...)
			assertExtraOperandFailure(t, result, tt.commandPath)

			hostAfter := snapshotOperandTree(t, fixture.host)
			sourceAfter := snapshotOperandTree(t, fixture.source)
			assertOperandSnapshotEqual(t, "host", hostAfter, hostBefore)
			assertOperandSnapshotEqual(t, "source", sourceAfter, sourceBefore)
		})
	}
}

func TestLinkExtraOperandPrecedesConfigDiscovery(t *testing.T) {
	bin := buildCubby(t)
	host := filepath.Join(t.TempDir(), "host-without-config")
	mustMkdir(t, host)

	result := runCubby(t, bin, host, "link", "unexpected")
	assertExtraOperandFailure(t, result, "cubby link")
	if combined := result.stdout + result.stderr; strings.Contains(combined, ".cubby.toml") {
		t.Fatalf("output %q contains a config discovery error, want argument validation first", combined)
	}
}

func prepareLinkOperandFixture(t *testing.T) operandFixture {
	fixture := prepareBaseOperandFixture(t)
	mustWrite(t, filepath.Join(fixture.source, "nvim", "init.work.lua"), "-- source file\n")
	return fixture
}

func prepareUnlinkOperandFixture(t *testing.T) operandFixture {
	fixture := prepareBaseOperandFixture(t)
	sourceFile := filepath.Join(fixture.source, "nvim", "init.work.lua")
	mustWrite(t, sourceFile, "-- source file\n")
	mustSymlink(t, filepath.Join(fixture.host, "nvim", "init.work.lua"), sourceFile)
	return fixture
}

func preparePruneOperandFixture(t *testing.T) operandFixture {
	fixture := prepareBaseOperandFixture(t)
	mustSymlink(t, filepath.Join(fixture.host, "nvim", "stale.work.lua"), filepath.Join(fixture.source, "nvim", "stale.work.lua"))
	return fixture
}

func prepareGitignoreSyncOperandFixture(t *testing.T) operandFixture {
	return prepareBaseOperandFixture(t)
}

func prepareBaseOperandFixture(t *testing.T) operandFixture {
	t.Helper()
	tmp := t.TempDir()
	fixture := operandFixture{
		host:   filepath.Join(tmp, "host"),
		source: filepath.Join(tmp, "source"),
	}
	mustWrite(t, filepath.Join(fixture.source, "cubby.toml"), "profiles = [\"work\"]\n")
	mustWrite(t, filepath.Join(fixture.host, ".cubby.toml"), fmt.Sprintf("profiles = [\"work\"]\n\n[[source]]\nname = \"source\"\npath = %q\n", fixture.source))
	return fixture
}

func snapshotOperandTree(t *testing.T, root string) []treeSnapshotEntry {
	t.Helper()
	var snapshot []treeSnapshotEntry
	err := filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entry := treeSnapshotEntry{
			path: filepath.ToSlash(relPath),
			kind: operandEntryKind(info.Mode()),
			perm: info.Mode().Perm(),
		}
		switch entry.kind {
		case "file":
			entry.content, err = os.ReadFile(path)
		case "symlink":
			entry.target, err = os.Readlink(path)
		}
		if err != nil {
			return err
		}
		snapshot = append(snapshot, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %q error = %v", root, err)
	}
	return snapshot
}

func operandEntryKind(mode fs.FileMode) string {
	switch {
	case mode.IsRegular():
		return "file"
	case mode.IsDir():
		return "directory"
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	default:
		return mode.Type().String()
	}
}

func assertExtraOperandFailure(t *testing.T, result runResult, commandPath string) {
	t.Helper()
	if result.code == 0 {
		t.Fatalf("exit code = 0, want argument failure; stdout = %q, stderr = %q", result.stdout, result.stderr)
	}
	combined := result.stdout + result.stderr
	for _, want := range []string{"unknown command", "unexpected", commandPath} {
		if !strings.Contains(combined, want) {
			t.Fatalf("argument error %q does not contain %q", combined, want)
		}
	}
}

func assertOperandSnapshotEqual(t *testing.T, tree string, got, want []treeSnapshotEntry) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s tree changed after rejected operand\ngot:  %#v\nwant: %#v", tree, got, want)
	}
}
