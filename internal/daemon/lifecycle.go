package daemon

import (
	"context"

	"krellin/internal/session"
)

// EnsureSessionFromHandshake creates or returns a session using repo_root.
func (d *Daemon) EnsureSessionFromHandshake(ctx context.Context, repoRoot string) (*session.Session, error) {
	return d.EnsureSessionForRepo(ctx, repoRoot)
}
