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

func (f *fakeClient) SendAction(ctx context.Context, action []byte) error  { return nil }
func (f *fakeClient) Subscribe(ctx context.Context) (<-chan []byte, error) { return f.events, nil }

func TestTUIRender(t *testing.T) {
	client := &fakeClient{events: make(chan []byte, 2)}
	model := newTUIModel(client, "s1", "agent")
	ev := protocol.Event{Type: protocol.EventTerminalOutput, Timestamp: time.Now(), Payload: mustJSON(protocol.TerminalOutputPayload{Stream: "stdout", Data: "ok\n"})}
	next, _ := model.Update(eventMsg{ev: ev})
	model = next.(*tuiModel)
	view := model.View().Content
	if !bytes.Contains([]byte(view), []byte("ok")) {
		t.Fatalf("expected terminal output in view, got %q", view)
	}
}

func mustJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
