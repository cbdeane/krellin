// Package config handles .krellinrc parsing and validation.
// Threat model (v0): config is user-controlled; parsing must fail closed and
// enforce digest pinning to avoid implicit tag drift.
package config
