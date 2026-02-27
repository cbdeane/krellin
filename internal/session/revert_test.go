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

const revertPatchPayload = `diff --git a/a.txt b/a.txt
index 1111111..2222222 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-hello
+hello world
`

func TestSessionHandlerRevert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	bk := patch.NewBookkeeper(dir)

	s := &Session{patches: bk, subscribers: map[chan protocol.Event]struct{}{}}
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.ApplyPatchPayload{Patch: revertPatchPayload})
	if err := h.Handle(context.Background(), protocol.Action{Type: protocol.ActionApplyPatch, Timestamp: time.Now(), Payload: payload}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if err := h.Handle(context.Background(), protocol.Action{Type: protocol.ActionRevert, Timestamp: time.Now()}); err != nil {
		t.Fatalf("revert: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}
