package daemon

import (
	"context"
	"testing"
	"time"

	"krellin/internal/protocol"
)

type noopHandler struct{}

func (n *noopHandler) Handle(ctx context.Context, action protocol.Action) error {
	return nil
}

func TestDaemonStartsSessions(t *testing.T) {
	d := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s1 := d.StartSession(ctx, "/repo1", "krellin-1", &noopHandler{})
	s2 := d.StartSession(ctx, "/repo2", "krellin-2", &noopHandler{})

	if s1.ID() == s2.ID() {
		t.Fatalf("expected distinct session IDs")
	}

	if d.SessionCount() != 2 {
		t.Fatalf("expected 2 sessions, got %d", d.SessionCount())
	}
}

func TestDaemonFanout(t *testing.T) {
	d := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := d.StartSession(ctx, "/repo", "krellin-1", &noopHandler{})
	ch, err := d.Subscribe(sess.ID(), 10)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	sess.Submit(protocol.Action{ActionID: "a1", SessionID: sess.ID(), AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()})

	select {
	case ev := <-ch:
		if ev.Type == "" {
			t.Fatalf("expected event")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for event")
	}
}
