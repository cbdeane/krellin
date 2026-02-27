// Package repo detects repo roots and derives stable repo identifiers.
// Threat model (v0): path inputs may be untrusted; normalization avoids
// collisions or ambiguity when generating repo IDs.
package repo
