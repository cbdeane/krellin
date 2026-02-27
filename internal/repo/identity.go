package repo

import (
	"path/filepath"
)

// ResolveRoot resolves and normalizes a repo root from a start path.
func ResolveRoot(start string) (string, error) {
	root, err := FindRoot(start)
	if err != nil {
		return "", err
	}
	return filepath.Abs(root)
}
