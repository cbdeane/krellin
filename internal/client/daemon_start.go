package client

import (
	"errors"
	"net"
	"os/exec"
	"time"
)

// EnsureDaemon starts krellind if the socket is not reachable.
func EnsureDaemon(sock string) error {
	conn, err := net.DialTimeout("unix", sock, 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return nil
	}
	cmd := exec.Command("krellind", "-sock", sock)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Wait briefly for socket to be ready.
	for i := 0; i < 10; i++ {
		conn, err = net.DialTimeout("unix", sock, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("daemon did not start")
}
