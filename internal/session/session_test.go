package session

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"krellin/internal/policy"
	"krellin/internal/protocol"
)

type noopHandler struct{}

func (n *noopHandler) Handle(ctx context.Context, action protocol.Action) error {
	return nil
}

func TestSessionStartEmits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := New(Options{
		SessionID:   "s1",
		RepoRoot:    "/repo",
		CapsuleName: "krellin-1",
		Handler:     &noopHandler{},
	})

	ch := sess.Subscribe(10)
	sess.Start(ctx)

	select {
	case ev := <-ch:
		if ev.Type != protocol.EventSessionStarted {
			t.Fatalf("expected session.started, got %q", ev.Type)
		}
		var payload protocol.SessionStartedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if payload.RepoRoot != "/repo" || payload.CapsuleName != "krellin-1" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for session.started")
	}
}

func TestSessionFanout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := New(Options{
		SessionID:   "s1",
		RepoRoot:    "/repo",
		CapsuleName: "krellin-1",
		Handler:     &noopHandler{},
	})

	ch1 := sess.Subscribe(10)
	ch2 := sess.Subscribe(10)
	sess.Start(ctx)

	sess.Submit(protocol.Action{ActionID: "a1", SessionID: "s1", AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()})

	got1 := waitForEvent(t, ch1)
	got2 := waitForEvent(t, ch2)

	if got1.Type == "" || got2.Type == "" {
		t.Fatalf("expected events on both channels")
	}
}

func TestSessionEnsureCapsuleOnStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	caps := &fakeCapsule{}
	sess := New(Options{
		SessionID:   "s1",
		RepoRoot:    "/repo",
		CapsuleName: "krellin-repo1",
		Handler:     &noopHandler{},
		Capsule:     caps,
		Policy:      policy.DefaultPolicy("/repo", "/home/user"),
		ImageDigest: "img@sha256:abc",
		NetworkOn:   true,
		CPUs:        2,
		MemoryMB:    4096,
	})

	sess.Start(ctx)
	if !caps.called {
		t.Fatalf("expected capsule ensure on start")
	}
}

func waitForEvent(t *testing.T, ch <-chan protocol.Event) protocol.Event {
	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for event")
	}
	return protocol.Event{}
}
