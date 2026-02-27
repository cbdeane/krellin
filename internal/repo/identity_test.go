package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRoot(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	root, err := ResolveRoot(repoRoot)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if root != repoRoot {
		t.Fatalf("expected %q, got %q", repoRoot, root)
	}
}
