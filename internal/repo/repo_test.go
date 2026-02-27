package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	nested := filepath.Join(repoRoot, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	root, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}
	if root != repoRoot {
		t.Fatalf("expected %q, got %q", repoRoot, root)
	}
}

func TestFindRepoRootWithGitFile(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "worktree")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitFile := filepath.Join(repoRoot, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /tmp/elsewhere"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	root, err := FindRoot(repoRoot)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}
	if root != repoRoot {
		t.Fatalf("expected %q, got %q", repoRoot, root)
	}
}

func TestRepoIDStable(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	id1, err := RepoID(repoRoot)
	if err != nil {
		t.Fatalf("repo id: %v", err)
	}
	id2, err := RepoID(repoRoot)
	if err != nil {
		t.Fatalf("repo id: %v", err)
	}

	if id1 == "" || id1 != id2 {
		t.Fatalf("expected stable non-empty repo id, got %q and %q", id1, id2)
	}
}

func TestFindRootError(t *testing.T) {
	dir := t.TempDir()
	_, err := FindRoot(dir)
	if err == nil {
		t.Fatalf("expected error when no repo root")
	}
}
