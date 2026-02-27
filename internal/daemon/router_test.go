package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"krellin/internal/protocol"
)

type routerHandler struct{}

func (n *routerHandler) Handle(ctx context.Context, action protocol.Action) error {
	return nil
}

func TestRouterServeConnRoutesActions(t *testing.T) {
	d := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := d.StartSession(ctx, "/repo", "krellin-1", &routerHandler{})
	router := NewRouter(d, NewTransport())

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		_ = router.ServeConn(ctx, server, "")
	}()

	tr := NewTransport()
	scanner := bufio.NewScanner(client)
	if err := WriteConnect(client, sess.ID(), "", true); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !scanner.Scan() {
		t.Fatalf("expected connected response")
	}
	var resp ConnectResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.SessionID != sess.ID() {
		t.Fatalf("unexpected session id")
	}
	action := protocol.Action{ActionID: "a1", SessionID: sess.ID(), AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()}
	if err := tr.SendAction(ctx, client, action); err != nil {
		t.Fatalf("send: %v", err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		t.Fatalf("expected event")
	}
	var ev protocol.Event
	if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.SessionID != sess.ID() {
		t.Fatalf("unexpected session id")
	}
	_ = client.Close()
	cancel()
}
