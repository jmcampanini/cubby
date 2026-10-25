// Package output formats paths, lists, and action reports for command output.
package output

import (
	"fmt"
	"path/filepath"
	"strings"
)

// QuotedList formats comma-separated, quoted strings.
func QuotedList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}

// SlashPath cleans a nonempty path and uses forward slashes.
func SlashPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}
