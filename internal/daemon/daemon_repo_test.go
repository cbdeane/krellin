package daemon

import (
	"context"
	"testing"
)

func TestEnsureSessionByRepo(t *testing.T) {
	d := New()
	ctx := context.Background()

	s1 := d.EnsureSession(ctx, "/repo", "krellin-1", &noopHandler{})
	s2 := d.EnsureSession(ctx, "/repo", "krellin-1", &noopHandler{})

	if s1.ID() != s2.ID() {
		t.Fatalf("expected same session for repo root")
	}
}
