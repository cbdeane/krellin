package daemon

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"syscall"
	"testing"
)

func TestServerAcceptsConnections(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "krellin.sock")

	srv := NewServer(sock)
	if err := srv.Start(context.Background()); err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("socket not permitted in test environment: %v", err)
		}
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()
}
