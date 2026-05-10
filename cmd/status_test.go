package cmd

import (
	"strings"
	"testing"
)

func TestStatusCommandExitsWithNotImplementedError(t *testing.T) {
	_, _, err := executeForTest("status")
	if err == nil {
		t.Fatalf("status error = nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("status error = %q, want not implemented", err)
	}
}
