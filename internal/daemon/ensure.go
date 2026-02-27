package daemon

import (
	"context"
	"fmt"
	"sync/atomic"

	"krellin/internal/repo"
	"krellin/internal/session"
)

// EnsureSessionForRepo resolves repo root and ensures a session exists.
func (d *Daemon) EnsureSessionForRepo(ctx context.Context, startPath string) (*session.Session, error) {
	root, err := repo.ResolveRoot(startPath)
	if err != nil {
		return nil, err
	}
	if d.factory != nil {
		sess, err := d.factory(ctx, root)
		if err != nil {
			return nil, err
		}
		if sess.ID() == "" {
			id := fmt.Sprintf("s-%d", atomic.AddUint64(&sessionCounter, 1))
			sess.SetID(id)
		}
		sess.Start(ctx)
		d.mu.Lock()
		d.sessions[sess.ID()] = sess
		d.byRepo[root] = sess.ID()
		d.mu.Unlock()
		return sess, nil
	}
	return d.EnsureSession(ctx, root, fmt.Sprintf("krellin-%s", "default"), nil), nil
}
