package session

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"krellin/internal/capsule"
	"krellin/internal/containers"
	"krellin/internal/patch"
	"krellin/internal/protocol"
)

type recordCapsule struct {
	lastNetwork *bool
	lastReset   *bool
	lastFreeze  string
	ptyConn     *recordPTYConn
}

func (r *recordCapsule) Ensure(ctx context.Context, cfg capsule.Config) (capsule.Handle, error) {
	return capsule.Handle{ID: "krellin-repo1"}, nil
}
func (r *recordCapsule) Start(ctx context.Context, handle capsule.Handle) error { return nil }
func (r *recordCapsule) Stop(ctx context.Context, handle capsule.Handle) error { return nil }
func (r *recordCapsule) Reset(ctx context.Context, handle capsule.Handle, imageDigest string, preserveVolumes bool) error {
	r.lastReset = &preserveVolumes
	return nil
}
func (r *recordCapsule) AttachPTY(ctx context.Context, handle capsule.Handle) (capsule.PTYConn, error) {
	if r.ptyConn == nil {
		r.ptyConn = &recordPTYConn{}
	}
	return r.ptyConn, nil
}
func (r *recordCapsule) Commit(ctx context.Context, handle capsule.Handle, opts capsule.CommitOptions) (string, error) {
	r.lastFreeze = opts.Mode
	return "sha256:abc", nil
}
func (r *recordCapsule) SetNetwork(ctx context.Context, handle capsule.Handle, enabled bool) error {
	r.lastNetwork = &enabled
	return nil
}
func (r *recordCapsule) Status(ctx context.Context, handle capsule.Handle) (capsule.Status, error) {
	return capsule.Status{}, nil
}

type recordPTYConn struct {
	writes []string
	reads  []string
}

func (r *recordPTYConn) Read(p []byte) (int, error) {
	if len(r.reads) == 0 {
		return 0, context.Canceled
	}
	next := r.reads[0]
	r.reads = r.reads[1:]
	copy(p, []byte(next))
	return len(next), nil
}
func (r *recordPTYConn) Write(p []byte) (int, error) { r.writes = append(r.writes, string(p)); return len(p), nil }
func (r *recordPTYConn) Close() error                { return nil }

func TestSessionHandlerNetworkToggle(t *testing.T) {
	caps := &recordCapsule{}
	s := newBareSession(t, caps, nil)
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionNetworkToggle,
		Timestamp: time.Now(),
		Payload:   []byte(`{"enabled":false}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if caps.lastNetwork == nil || *caps.lastNetwork != false {
		t.Fatalf("expected network disabled")
	}
}

func TestSessionHandlerRunCommand(t *testing.T) {
	caps := &recordCapsule{}
	s := newBareSession(t, caps, nil)
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionRunCommand,
		Timestamp: time.Now(),
		Payload:   []byte(`{"command":"echo hi"}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if caps.ptyConn == nil || len(caps.ptyConn.writes) == 0 {
		t.Fatalf("expected write to pty")
	}
	if caps.ptyConn.writes[0] != "echo hi\n" {
		t.Fatalf("unexpected write: %q", caps.ptyConn.writes[0])
	}
}

func TestSessionHandlerFreeze(t *testing.T) {
	caps := &recordCapsule{}
	resolver := &fakeResolver{}
	updater := &fakeUpdater{}
	publisher := &fakePublisher{}
	s := &Session{
		capsule:     caps,
		handle:      capsule.Handle{ID: "krellin-repo1"},
		resolver:    resolver,
		updater:     updater,
		configPath:  "/tmp/.krellinrc",
		publisher:   publisher,
		publishTo:   "ghcr.io/acme/app:latest",
		platforms:   []string{"linux/amd64", "linux/arm64"},
		subscribers: map[chan protocol.Event]struct{}{},
	}
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionFreeze,
		Timestamp: time.Now(),
		Payload:   []byte(`{"mode":"clean"}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if caps.lastFreeze != "clean" {
		t.Fatalf("expected freeze mode clean, got %q", caps.lastFreeze)
	}
	if updater.lastDigest != "repo@sha256:abc" {
		t.Fatalf("expected updater digest, got %q", updater.lastDigest)
	}
	if publisher.lastTarget != "ghcr.io/acme/app:latest" {
		t.Fatalf("expected publish target")
	}
	if resolver.lastRef != "ghcr.io/acme/app@sha256:abc" {
		t.Fatalf("expected resolver to see published ref, got %q", resolver.lastRef)
	}
}

func TestSessionHandlerResetPreserve(t *testing.T) {
	caps := &recordCapsule{}
	s := &Session{
		capsule:     caps,
		handle:      capsule.Handle{ID: "krellin-repo1"},
		subscribers: map[chan protocol.Event]struct{}{},
	}
	caps.ptyConn = &recordPTYConn{reads: []string{"ready"}}
	ch := s.Subscribe(10)
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionReset,
		Timestamp: time.Now(),
		Payload:   []byte(`{"preserve_volumes":false}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if caps.lastReset == nil || *caps.lastReset != false {
		t.Fatalf("expected preserve_volumes=false")
	}
	foundReset := false
	foundBanner := false
	for i := 0; i < 3; i++ {
		select {
		case ev := <-ch:
			if ev.Type == protocol.EventResetCompleted {
				foundReset = true
			}
			if ev.Type == protocol.EventTerminalOutput {
				foundBanner = true
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for reset events")
		}
	}
	if !foundReset || !foundBanner {
		t.Fatalf("expected reset.completed and terminal.output")
	}
}

func TestSessionHandlerContainersListEmitsEvent(t *testing.T) {
	caps := &recordCapsule{}
	s := newBareSession(t, caps, fakeInventory{})
	ch := s.Subscribe(1)
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionContainersList,
		Timestamp: time.Now(),
		Payload:   []byte(`{}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	select {
	case ev := <-ch:
		if ev.Type != protocol.EventContainersList {
			t.Fatalf("unexpected event type: %q", ev.Type)
		}
		var payload protocol.ContainersListPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if len(payload.Capsules) != 1 || payload.Capsules[0].RepoID != "repo1" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for containers list event")
	}
}

func TestSessionHandlerRunCommandEmitsTerminalOutput(t *testing.T) {
	caps := &recordCapsule{ptyConn: &recordPTYConn{reads: []string{"ok"}}}
	s := newBareSession(t, caps, nil)
	ch := s.Subscribe(10)
	h := SessionHandler{Session: s}
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionRunCommand,
		Timestamp: time.Now(),
		Payload:   []byte(`{"command":"echo hi"}`),
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	for {
		select {
		case ev := <-ch:
			if ev.Type == protocol.EventTerminalOutput {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for terminal output event")
		}
	}
}

type fakeInventory struct{}

func (f fakeInventory) ListCapsules(ctx context.Context) ([]containers.CapsuleInfo, error) {
	return []containers.CapsuleInfo{{ID: "id1", Name: "krellin-repo1", RepoID: "repo1"}}, nil
}

func (f fakeInventory) ListImages(ctx context.Context) ([]containers.ImageInfo, error) {
	return []containers.ImageInfo{{ID: "img1", Repository: "repo", Tag: "tag"}}, nil
}

type fakeResolver struct {
	lastRef string
}

func (f *fakeResolver) ResolveDigest(ctx context.Context, imageRef string) (string, error) {
	f.lastRef = imageRef
	return "repo@sha256:abc", nil
}

type fakeUpdater struct {
	lastDigest string
}

func (f *fakeUpdater) UpdateImage(path string, digest string) error {
	f.lastDigest = digest
	return nil
}

type fakePublisher struct {
	lastImage     string
	lastTarget    string
	lastPlatforms []string
}

func (f *fakePublisher) Push(ctx context.Context, imageRef string, target string, platforms []string) (string, error) {
	f.lastImage = imageRef
	f.lastTarget = target
	f.lastPlatforms = platforms
	return "ghcr.io/acme/app@sha256:abc", nil
}

func newBareSession(t *testing.T, caps capsule.Capsule, inventory ContainersInventory) *Session {
	t.Helper()
	return &Session{
		capsule:     caps,
		handle:      capsule.Handle{ID: "krellin-repo1"},
		subscribers: map[chan protocol.Event]struct{}{},
		inventory:   inventory,
		patches:     patch.NewBookkeeper(t.TempDir()),
	}
}
