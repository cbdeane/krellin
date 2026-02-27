// Package patch applies unified diffs and tracks revert state.
// Threat model (v0): patches may be malformed; apply must fail safely and never
// partially apply changes without bookkeeping.
package patch
