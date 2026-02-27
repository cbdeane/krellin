package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net"

	"krellin/internal/daemon"
)

type SocketClient struct {
	addr      string
	sessionID string
	repoRoot  string
}

func NewSocketClient(addr string, sessionID string, repoRoot string) *SocketClient {
	return &SocketClient{addr: addr, sessionID: sessionID, repoRoot: repoRoot}
}

func (c *SocketClient) SendAction(ctx context.Context, action []byte) error {
	conn, err := net.Dial("unix", c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := daemon.WriteConnect(conn, c.sessionID, c.repoRoot); err != nil {
		return err
	}
	if sessionID, err := readConnectResponse(conn); err == nil && sessionID != "" && c.sessionID == "" {
		c.sessionID = sessionID
	}
	_, err = conn.Write(append(action, '\n'))
	return err
}

func (c *SocketClient) Subscribe(ctx context.Context) (<-chan []byte, error) {
	conn, err := net.Dial("unix", c.addr)
	if err != nil {
		return nil, err
	}
	if err := daemon.WriteConnect(conn, c.sessionID, c.repoRoot); err != nil {
		return nil, err
	}
	sessionID, err := readConnectResponse(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if sessionID != "" && c.sessionID == "" {
		c.sessionID = sessionID
	}
	out := make(chan []byte, 16)
	go func() {
		defer conn.Close()
		defer close(out)
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := append([]byte{}, scanner.Bytes()...)
			out <- line
		}
	}()
	return out, nil
}

func readConnectResponse(conn net.Conn) (string, error) {
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	var resp daemon.ConnectResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return "", err
	}
	return resp.SessionID, nil
}
