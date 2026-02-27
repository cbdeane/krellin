package executor

import (
	"context"
	"sync"
	"testing"
	"time"

	"krellin/internal/protocol"
	"krellin/internal/queue"
)

type recordingEmitter struct {
	mu     sync.Mutex
	events []protocol.Event
}

func (r *recordingEmitter) Emit(event protocol.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recordingEmitter) Types() []protocol.EventType {
	r.mu.Lock()
	defer r.mu.Unlock()
	types := make([]protocol.EventType, 0, len(r.events))
	for _, ev := range r.events {
		types = append(types, ev.Type)
	}
	return types
}

type blockingHandler struct {
	mu       sync.Mutex
	inFlight bool
	started  chan string
	release  chan struct{}
}

func (h *blockingHandler) Handle(ctx context.Context, action protocol.Action) error {
	h.mu.Lock()
	if h.inFlight {
		h.mu.Unlock()
		return nil
	}
	h.inFlight = true
	h.mu.Unlock()

	h.started <- action.ActionID
	select {
	case <-h.release:
	case <-ctx.Done():
	}

	h.mu.Lock()
	h.inFlight = false
	h.mu.Unlock()
	return nil
}

func TestExecutorSerializesActions(t *testing.T) {
	q := queue.New[protocol.Action]()
	emitter := &recordingEmitter{}
	handler := &blockingHandler{
		started: make(chan string, 2),
		release: make(chan struct{}),
	}

	ex := New(q, handler, emitter)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ex.Run(ctx)

	q.Enqueue(protocol.Action{ActionID: "a1", SessionID: "s", AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()})
	q.Enqueue(protocol.Action{ActionID: "a2", SessionID: "s", AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()})

	first := <-handler.started
	if first != "a1" {
		t.Fatalf("expected a1 first, got %q", first)
	}

	select {
	case second := <-handler.started:
		t.Fatalf("unexpected second start before release: %q", second)
	case <-time.After(50 * time.Millisecond):
	}

	handler.release <- struct{}{}
	second := <-handler.started
	if second != "a2" {
		t.Fatalf("expected a2 second, got %q", second)
	}
	handler.release <- struct{}{}
}

func TestExecutorEmitsEvents(t *testing.T) {
	q := queue.New[protocol.Action]()
	emitter := &recordingEmitter{}
	handler := &blockingHandler{
		started: make(chan string, 1),
		release: make(chan struct{}),
	}

	ex := New(q, handler, emitter)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ex.Run(ctx)
	q.Enqueue(protocol.Action{ActionID: "a1", SessionID: "s", AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()})

	<-handler.started
	handler.release <- struct{}{}

	// Allow executor loop to emit events.
	time.Sleep(20 * time.Millisecond)
	types := emitter.Types()

	if len(types) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(types))
	}
	if types[0] != protocol.EventExecutorBusy || types[1] != protocol.EventActionStarted || types[2] != protocol.EventActionFinished || types[3] != protocol.EventExecutorIdle {
		t.Fatalf("unexpected event order: %v", types)
	}
}
