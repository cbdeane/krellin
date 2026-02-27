// Package policy enforces safety defaults (mounts, network, resources).
// Threat model (v0): host mounts are dangerous; reject all mounts outside
// repo root unless /unsafe is explicitly enabled.
package policy
