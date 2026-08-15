package cmd

import (
	"bytes"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestApplicationOwnedRunnableLeavesRejectExtraOperands(t *testing.T) {
	tests := []struct {
		path      string
		operands  []string
		wantError string
	}{
		{path: "link", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "unlink", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "prune", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "status", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "doctor", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "gitignore check", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "gitignore sync", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "profile list", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "profile effective", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "source list", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "config", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "lazygit", operands: []string{"unexpected"}, wantError: `unknown command "unexpected"`},
		{path: "docs", operands: []string{"manual", "unexpected"}, wantError: "accepts at most one argument"},
	}

	root := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{})
	gotInventory := applicationRunnableLeafPaths(t, root)
	wantInventory := make([]string, 0, len(tests))
	for _, tt := range tests {
		wantInventory = append(wantInventory, tt.path)
	}
	sort.Strings(wantInventory)
	if !slices.Equal(gotInventory, wantInventory) {
		t.Fatalf("application runnable leaf inventory = %q, want %q", gotInventory, wantInventory)
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.path, " ", "/"), func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer
			root := NewRootCommand(&out, &errOut)
			leaf, remaining, err := root.Find(strings.Fields(tt.path))
			if err != nil {
				t.Fatalf("find %q: %v", tt.path, err)
			}
			if len(remaining) != 0 || leaf.CommandPath() != "cubby "+tt.path {
				t.Fatalf("find %q = %q with remaining args %q", tt.path, leaf.CommandPath(), remaining)
			}

			runnerCalled := false
			leaf.Run = nil
			leaf.RunE = func(*cobra.Command, []string) error {
				runnerCalled = true
				return nil
			}

			args := append(strings.Fields(tt.path), tt.operands...)
			root.SetArgs(args)
			err = root.Execute()
			if err == nil {
				t.Fatal("Execute() error = nil, want argument error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Execute() error = %q, want substring %q", err, tt.wantError)
			}
			if runnerCalled {
				t.Fatal("leaf runner was called before argument validation")
			}
		})
	}
}

func TestCommandGroupsRejectExtraOperandsAndPreserveBareHelp(t *testing.T) {
	tests := []struct {
		path     string
		helpText string
	}{
		{path: "profile", helpText: "effective"},
		{path: "source", helpText: "list"},
		{path: "gitignore", helpText: "check"},
	}

	for _, tt := range tests {
		t.Run(tt.path+"/surplus_operand", func(t *testing.T) {
			root := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{})
			group, remaining, err := root.Find([]string{tt.path})
			if err != nil {
				t.Fatalf("find %q: %v", tt.path, err)
			}
			if len(remaining) != 0 || group.CommandPath() != "cubby "+tt.path {
				t.Fatalf("find %q = %q with remaining args %q", tt.path, group.CommandPath(), remaining)
			}

			runnerCalled := false
			group.Run = nil
			group.RunE = func(*cobra.Command, []string) error {
				runnerCalled = true
				return nil
			}

			root.SetArgs([]string{tt.path, "unexpected"})
			err = root.Execute()
			if err == nil {
				t.Fatal("Execute() error = nil, want argument error")
			}
			if !strings.Contains(err.Error(), `unknown command "unexpected"`) || !strings.Contains(err.Error(), "cubby "+tt.path) {
				t.Fatalf("Execute() error = %q, want actionable %s argument error", err, tt.path)
			}
			if runnerCalled {
				t.Fatal("group runner was called before argument validation")
			}
		})

		t.Run(tt.path+"/bare_help", func(t *testing.T) {
			var out bytes.Buffer
			root := NewRootCommand(&out, &bytes.Buffer{})
			root.SetArgs([]string{tt.path})

			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(out.String(), tt.helpText) {
				t.Fatalf("bare %s help does not contain %q:\n%s", tt.path, tt.helpText, out.String())
			}
		})
	}
}

func TestGeneratedCompletionOperands(t *testing.T) {
	t.Run("valid shell", func(t *testing.T) {
		var out bytes.Buffer
		root := NewRootCommand(&out, &bytes.Buffer{})
		root.SetArgs([]string{"completion", "bash"})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !strings.Contains(out.String(), "__start_cubby") {
			t.Fatalf("completion output does not contain the cubby bash completion entry point:\n%s", out.String())
		}
	})

	t.Run("surplus operand", func(t *testing.T) {
		var out bytes.Buffer
		root := NewRootCommand(&out, &bytes.Buffer{})
		root.SetArgs([]string{"completion", "bash", "unexpected"})

		err := root.Execute()
		if err == nil {
			t.Fatal("Execute() error = nil, want argument error")
		}
		if !strings.Contains(err.Error(), `unknown command "unexpected"`) {
			t.Fatalf("Execute() error = %q, want completion argument error", err)
		}
	})
}

func applicationRunnableLeafPaths(t *testing.T, root *cobra.Command) []string {
	t.Helper()

	var paths []string
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		children := command.Commands()
		if command.Runnable() && command.Args == nil {
			t.Errorf("application runnable command %q has no Args validator", command.CommandPath())
		}
		if len(children) == 0 && command.Runnable() {
			paths = append(paths, strings.TrimPrefix(command.CommandPath(), root.CommandPath()+" "))
			return
		}
		for _, child := range children {
			visit(child)
		}
	}
	for _, child := range root.Commands() {
		visit(child)
	}
	sort.Strings(paths)
	return paths
}
