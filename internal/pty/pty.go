package pty

import "io"

// Conn represents a PTY connection to a running capsule.
type Conn interface {
	io.Reader
	io.Writer
	io.Closer
}
