package daemon

import (
	"context"
	"net"
	"time"

	"krellin/internal/protocol"
)

type Router struct {
	daemon    *Daemon
	transport *Transport
}

func NewRouter(d *Daemon, t *Transport) *Router {
	return &Router{daemon: d, transport: t}
}

func (r *Router) ServeConn(ctx context.Context, conn net.Conn, sessionID string) error {
	var repoRoot string
	if sessionID == "" {
		var err error
		sessionID, repoRoot, err = ReadConnect(conn)
		if err != nil {
			return err
		}
		if sessionID == "" && repoRoot != "" {
			sess, err := r.daemon.EnsureSessionFromHandshake(ctx, repoRoot)
			if err != nil {
				_ = r.transport.SendEvent(ctx, conn, errorEvent("", "", err.Error()))
				return err
			}
			sessionID = sess.ID()
		}
		if sessionID == "" {
			return errSessionNotFound
		}
	}
	_ = WriteConnectResponse(conn, sessionID)
	// Emit a direct connected message so clients can confirm the stream.
	connectedPayload, _ := protocol.MarshalPayload(protocol.AgentMessagePayload{Content: "connected"})
	_ = r.transport.SendEvent(ctx, conn, protocol.Event{
		EventID:   "connected",
		SessionID: sessionID,
		Timestamp: time.Now().UTC(),
		Type:      protocol.EventAgentMessage,
		Source:    protocol.SourceSystem,
		Payload:   connectedPayload,
	})

	events, err := r.daemon.Subscribe(sessionID, 100)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sendErr := make(chan error, 1)
	go func() {
		for ev := range events {
			if err := r.transport.SendEvent(ctx, conn, ev); err != nil {
				sendErr <- err
				return
			}
		}
		sendErr <- nil
	}()

	actionCh := make(chan protocol.Action, 8)
	go func() {
		_ = r.transport.ReadActions(ctx, conn, actionCh)
		close(actionCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sendErr:
			return err
		case action, ok := <-actionCh:
			if !ok {
				return nil
			}
			if action.SessionID == "" {
				action.SessionID = sessionID
			}
			sess := r.daemon.Session(action.SessionID)
			if sess == nil {
				return errSessionNotFound
			}
			_ = sess.Submit(action)
		}
	}
}
