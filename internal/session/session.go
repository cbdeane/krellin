package session

import (
	"context"
	"strings"
	"sync"
	"time"

	"krellin/internal/capsule"
	"krellin/internal/executor"
	"krellin/internal/patch"
	"krellin/internal/policy"
	"krellin/internal/protocol"
	"krellin/internal/queue"
)

type Options struct {
	SessionID   string
	RepoRoot    string
	CapsuleName string
	Handler     executor.Handler
	Capsule     capsule.Capsule
	Policy      policy.Policy
	ImageDigest string
	NetworkOn   bool
	CPUs        int
	MemoryMB    int
	Inventory   ContainersInventory
	Patches     *patch.Bookkeeper
	ConfigPath  string
	Resolver    ImageResolver
	Updater     ConfigUpdater
	Publisher   ImagePublisher
	PublishTo   string
	Platforms   []string
}

type Session struct {
	id          string
	repoRoot    string
	capsuleName string
	queue       *queue.Queue[protocol.Action]
	executor    *executor.Executor
	subscribers map[chan protocol.Event]struct{}
	mu          sync.Mutex
	startOnce   sync.Once
	ptyOnce     sync.Once
	capsule     capsule.Capsule
	policy      policy.Policy
	imageDigest string
	networkOn   bool
	cpus        int
	memoryMB    int
	handle      capsule.Handle
	pty         capsule.PTYConn
	inventory   ContainersInventory
	patches     *patch.Bookkeeper
	configPath  string
	resolver    ImageResolver
	updater     ConfigUpdater
	publisher   ImagePublisher
	publishTo   string
	platforms   []string
}

func New(opts Options) *Session {
	q := queue.New[protocol.Action]()
	s := &Session{
		id:          opts.SessionID,
		repoRoot:    opts.RepoRoot,
		capsuleName: opts.CapsuleName,
		queue:       q,
		subscribers: map[chan protocol.Event]struct{}{},
		capsule:     opts.Capsule,
		policy:      opts.Policy,
		imageDigest: opts.ImageDigest,
		networkOn:   opts.NetworkOn,
		cpus:        opts.CPUs,
		memoryMB:    opts.MemoryMB,
		inventory:   opts.Inventory,
	patches:     opts.Patches,
	configPath:  opts.ConfigPath,
	resolver:    opts.Resolver,
	updater:     opts.Updater,
	publisher:   opts.Publisher,
	publishTo:   opts.PublishTo,
	platforms:   opts.Platforms,
	}
	handler := opts.Handler
	if handler == nil {
		handler = SessionHandler{Session: s}
	}
	s.executor = executor.New(q, handler, s)
	return s
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) SetID(id string) {
	s.id = id
}

func (s *Session) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		if s.capsule != nil {
			manager := CapsuleManager{Capsule: s.capsule, Policy: s.policy}
			handle, _ := manager.Ensure(ctx, capsule.Config{
				RepoID:      strings.TrimPrefix(s.capsuleName, "krellin-"),
				RepoRoot:    s.repoRoot,
				ImageDigest: s.imageDigest,
				NetworkOn:   s.networkOn,
				CPUs:        s.cpus,
				MemoryMB:    s.memoryMB,
			})
			s.handle = handle
		}
		payload, _ := protocol.MarshalPayload(protocol.SessionStartedPayload{RepoRoot: s.repoRoot, CapsuleName: s.capsuleName})
		s.Emit(protocol.Event{
			EventID:   "session-started",
			SessionID: s.id,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventSessionStarted,
			Source:    protocol.SourceSystem,
			Payload:   payload,
		})

		go s.executor.Run(ctx)
	})
}

func (s *Session) Submit(action protocol.Action) error {
	if action.SessionID == "" {
		action.SessionID = s.id
	}
	return s.queue.Enqueue(action)
}

func (s *Session) Subscribe(buffer int) chan protocol.Event {
	ch := make(chan protocol.Event, buffer)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *Session) Unsubscribe(ch chan protocol.Event) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	s.mu.Unlock()
	close(ch)
}

// Emit implements executor.Emitter.
func (s *Session) Emit(event protocol.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Session) ensurePTY(ctx context.Context) error {
	if s.capsule == nil {
		return nil
	}
	var err error
	s.ptyOnce.Do(func() {
		var conn capsule.PTYConn
		conn, err = s.capsule.AttachPTY(ctx, s.handle)
		if err != nil {
			return
		}
		s.pty = conn
		go s.streamPTY(ctx, conn)
	})
	return err
}

func (s *Session) resetPTY() {
	if s.pty != nil {
		_ = s.pty.Close()
		s.pty = nil
	}
	s.ptyOnce = sync.Once{}
}

func (s *Session) streamPTY(ctx context.Context, conn capsule.PTYConn) {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			payload, _ := protocol.MarshalPayload(protocol.TerminalOutputPayload{
				Stream: "stdout",
				Data:   string(buf[:n]),
			})
			s.Emit(protocol.Event{
				EventID:   "terminal-output",
				SessionID: s.id,
				Timestamp: time.Now().UTC(),
				Type:      protocol.EventTerminalOutput,
				Source:    protocol.SourceExecutor,
				Payload:   payload,
			})
		}
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
