package daemon

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"krellin/internal/executor"
	"krellin/internal/protocol"
	"krellin/internal/session"
)

var errSessionNotFound = errors.New("session not found")

type Daemon struct {
	mu       sync.Mutex
	sessions map[string]*session.Session
	byRepo   map[string]string
	factory  SessionFactory
}

func New() *Daemon {
	return &Daemon{sessions: map[string]*session.Session{}, byRepo: map[string]string{}}
}

type SessionFactory func(ctx context.Context, repoRoot string) (*session.Session, error)

func (d *Daemon) SetFactory(factory SessionFactory) {
	d.factory = factory
}

var sessionCounter uint64

func (d *Daemon) StartSession(ctx context.Context, repoRoot, capsuleName string, handler executor.Handler) *session.Session {
	id := fmt.Sprintf("s-%d", atomic.AddUint64(&sessionCounter, 1))
	sess := session.New(session.Options{
		SessionID:   id,
		RepoRoot:    repoRoot,
		CapsuleName: capsuleName,
		Handler:     handler,
	})
	sess.Start(ctx)

	d.mu.Lock()
	d.sessions[id] = sess
	if repoRoot != "" {
		d.byRepo[repoRoot] = id
	}
	d.mu.Unlock()
	return sess
}

func (d *Daemon) Subscribe(sessionID string, buffer int) (chan protocol.Event, error) {
	d.mu.Lock()
	sess := d.sessions[sessionID]
	d.mu.Unlock()
	if sess == nil {
		return nil, errSessionNotFound
	}
	return sess.Subscribe(buffer), nil
}

func (d *Daemon) SessionCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.sessions)
}

func (d *Daemon) Session(id string) *session.Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sessions[id]
}

func (d *Daemon) SessionByRepo(repoRoot string) *session.Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := d.byRepo[repoRoot]
	return d.sessions[id]
}

func (d *Daemon) EnsureSession(ctx context.Context, repoRoot, capsuleName string, handler executor.Handler) *session.Session {
	if repoRoot != "" {
		if sess := d.SessionByRepo(repoRoot); sess != nil {
			return sess
		}
	}
	return d.StartSession(ctx, repoRoot, capsuleName, handler)
}
