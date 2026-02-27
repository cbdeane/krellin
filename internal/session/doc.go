// Package session manages per-repo sessions and client subscriptions.
// Threat model (v0): clients are untrusted; event fanout must avoid blocking
// or crashing the session when subscribers misbehave.
package session
