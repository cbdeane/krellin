// Package pty defines PTY interfaces for streaming terminal I/O.
// Threat model (v0): PTY output may contain escape sequences; clients must
// render safely to prevent injection.
package pty
