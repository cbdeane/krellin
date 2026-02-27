package client

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"krellin/internal/protocol"
)

type fakeClient struct {
	events chan []byte
}

func (f *fakeClient) SendAction(ctx context.Context, action []byte) error { return nil }
func (f *fakeClient) Subscribe(ctx context.Context) (<-chan []byte, error) { return f.events, nil }

func TestTUIRender(t *testing.T) {
	buf := &bytes.Buffer{}
	client := &fakeClient{events: make(chan []byte, 2)}
	tui := NewTUI(client, buf, bytes.NewBuffer(nil), "s1", "agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = tui.Run(ctx)
	}()

	ev := protocol.Event{Type: protocol.EventTerminalOutput, Timestamp: time.Now(), Payload: mustJSON(protocol.TerminalOutputPayload{Stream: "stdout", Data: "ok\n"})}
	client.events <- mustJSON(ev)
	time.Sleep(20 * time.Millisecond)
	cancel()

	if !bytes.Contains(buf.Bytes(), []byte("Timeline")) {
		t.Fatalf("expected timeline in output")
	}
}

func mustJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
