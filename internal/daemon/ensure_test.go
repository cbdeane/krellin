package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSessionForRepo(t *testing.T) {
	d := New()
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	sess, err := d.EnsureSessionForRepo(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if sess == nil {
		t.Fatalf("expected session")
	}
}
