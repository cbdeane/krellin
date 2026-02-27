package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"krellin/internal/patch"
	"krellin/internal/protocol"
)

const patchPayload = `diff --git a/a.txt b/a.txt
index 1111111..2222222 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-hello
+hello world
`

func TestSessionHandlerApplyPatchEmitsDiff(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	bk := patch.NewBookkeeper(dir)
	s := &Session{patches: bk, subscribers: map[chan protocol.Event]struct{}{}}
	ch := s.Subscribe(10)
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.ApplyPatchPayload{Patch: patchPayload})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionApplyPatch,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Type != protocol.EventDiffReady {
			t.Fatalf("expected diff.ready, got %q", ev.Type)
		}
		var diffPayload protocol.DiffReadyPayload
		if err := json.Unmarshal(ev.Payload, &diffPayload); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if len(diffPayload.Files) != 1 || diffPayload.Files[0] != "a.txt" {
			t.Fatalf("unexpected files: %+v", diffPayload.Files)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for diff.ready")
	}
}
