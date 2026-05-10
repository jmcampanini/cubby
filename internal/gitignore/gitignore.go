package gitignore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// RequiredPatterns returns the .gitignore patterns required for the given profiles.
func RequiredPatterns(profiles []string) []string {
	patterns := make([]string, 0, len(profiles)*2)
	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		if profile == "" {
			continue
		}
		patterns = append(patterns, "*."+profile+".*", "*."+profile)
	}
	return patterns
}

// MissingPatterns returns required patterns that are not present as exact, trimmed, non-comment lines.
func MissingPatterns(content []byte, required []string) []string {
	existing := existingPatternSet(content)
	missing := make([]string, 0)
	for _, pattern := range required {
		if _, ok := existing[pattern]; !ok {
			missing = append(missing, pattern)
		}
	}
	return missing
}

// MissingPatternsFile reads path as a .gitignore file and returns absent required patterns.
// A missing file is treated as empty.
func MissingPatternsFile(path string, required []string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MissingPatterns(nil, required), nil
		}
		return nil, err
	}
	return MissingPatterns(content, required), nil
}

// AppendMissing appends missing patterns to path, creating the file if needed.
func AppendMissing(path string, missing []string) error {
	if len(missing) == 0 {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var buf bytes.Buffer
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		buf.WriteByte('\n')
	}
	for _, pattern := range missing {
		buf.WriteString(pattern)
		buf.WriteByte('\n')
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(buf.Bytes())
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func existingPatternSet(content []byte) map[string]struct{} {
	set := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = struct{}{}
	}
	return set
}
