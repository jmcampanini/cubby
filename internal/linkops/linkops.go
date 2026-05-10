package linkops

import (
	"os"
	"path/filepath"
)

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
