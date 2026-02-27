// Package executor serializes action execution and emits events.
// Threat model (v0): action handlers may fail; executor must emit failure
// events and never run actions concurrently.
package executor
