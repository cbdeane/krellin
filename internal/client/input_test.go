package client

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

type recordClient struct {
	last []byte
}

func (r *recordClient) SendAction(ctx context.Context, action []byte) error { r.last = action; return nil }
func (r *recordClient) Subscribe(ctx context.Context) (<-chan []byte, error) { return make(chan []byte), nil }

func TestInputSendsAction(t *testing.T) {
	buf := bytes.NewBufferString("echo hi\n")
	rc := &recordClient{}
	in := NewInput(rc, buf, bytes.NewBuffer(nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := in.Run(ctx, "s1", "agent"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(rc.last) == 0 {
		t.Fatalf("expected action payload")
	}
	var decoded map[string]any
	if err := json.Unmarshal(rc.last, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["type"] == nil {
		t.Fatalf("expected type field")
	}
}
