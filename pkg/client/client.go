package client

import "context"

// Client is a placeholder for the daemon client.
type Client interface {
	SendAction(ctx context.Context, action []byte) error
	Subscribe(ctx context.Context) (<-chan []byte, error)
}
