// Package app wires concrete adapters and services.
// Threat model (v0): wiring must preserve safety defaults and prevent adapters
// from bypassing policy enforcement.
package app
