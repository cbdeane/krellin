package daemon

import (
	"context"
	"net"
	"testing"
	"time"

	"krellin/internal/protocol"
)

func TestTransportSendAndReadAction(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	tr := NewTransport()
	out := make(chan protocol.Action, 1)

	go func() {
		_ = tr.ReadActions(context.Background(), c2, out)
	}()

	action := protocol.Action{ActionID: "a1", SessionID: "s1", AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()}
	if err := tr.SendAction(context.Background(), c1, action); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case got := <-out:
		if got.ActionID != "a1" {
			t.Fatalf("unexpected action: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for action")
	}
}
