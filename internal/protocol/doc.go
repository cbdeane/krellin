// Package protocol defines the action and event wire types for Krellin.
// Threat model (v0): input is untrusted from clients; validation must reject
// unknown types to prevent execution of unsupported operations.
package protocol
