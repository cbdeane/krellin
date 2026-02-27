// Package capsule defines the capsule abstraction used by the session executor.
// Threat model (v0): adapter implementations must not accept unsafe mounts or
// privilege escalation without explicit policy approval.
package capsule
