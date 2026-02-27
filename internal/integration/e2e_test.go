package integration

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"krellin/internal/daemon"
	"krellin/internal/protocol"
)

type noopHandler struct{}

func (n *noopHandler) Handle(ctx context.Context, action protocol.Action) error {
	return nil
}

func TestDaemonSocketE2E(t *testing.T) {
	if os.Getenv("KRELLIN_E2E") != "1" {
		t.Skip("KRELLIN_E2E not set")
	}

	dir := t.TempDir()
	sock := filepath.Join(dir, "krellin.sock")

	d := daemon.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := d.StartSession(ctx, "/repo", "krellin-1", &noopHandler{})
	router := daemon.NewRouter(d, daemon.NewTransport())
	srv := daemon.NewServerWithRouter(sock, router)
	if err := srv.Start(ctx); err != nil {
		if os.IsPermission(err) {
			t.Skipf("socket not permitted: %v", err)
		}
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("unix", sock)
	if err != nil {
		if os.IsPermission(err) {
			t.Skipf("socket not permitted: %v", err)
		}
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := daemon.WriteConnect(conn, sess.ID(), "", true); err != nil {
		t.Fatalf("connect: %v", err)
	}

	decoder := json.NewDecoder(conn)
	var resp daemon.ConnectResponse
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	action := protocol.Action{ActionID: "a1", SessionID: sess.ID(), AgentID: "agent", Type: protocol.ActionRunCommand, Timestamp: time.Now()}
	payload, _ := json.Marshal(action)
	if _, err := conn.Write(append(payload, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var ev protocol.Event
	if err := decoder.Decode(&ev); err != nil {
		t.Fatalf("event decode: %v", err)
	}
	if ev.Type == "" {
		t.Fatalf("expected event")
	}
}
