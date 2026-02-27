package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"

	"krellin/internal/protocol"
)

// Transport handles JSON line-delimited Actions/Events over a net.Conn.
type Transport struct{}

func NewTransport() *Transport {
	return &Transport{}
}

func (t *Transport) SendAction(ctx context.Context, conn net.Conn, action protocol.Action) error {
	data, err := json.Marshal(action)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}

func (t *Transport) ReadActions(ctx context.Context, conn net.Conn, out chan<- protocol.Action) error {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var action protocol.Action
		if err := json.Unmarshal(scanner.Bytes(), &action); err != nil {
			return err
		}
		out <- action
	}
	return scanner.Err()
}

func (t *Transport) SendEvent(ctx context.Context, conn net.Conn, event protocol.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}
