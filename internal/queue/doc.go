// Package queue provides a blocking FIFO queue for actions.
// Threat model (v0): callers may abandon contexts; queue should unblock on
// context cancellation to avoid goroutine leaks.
package queue
