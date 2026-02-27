# CLI (v0)

## krellin

Starts the TUI client and connects to the local daemon over the unix socket.

Flags:
- `-repo` repo root (defaults to cwd)
- `-sock` unix socket (defaults to `/tmp/krellin.sock`)

## krellind

Starts the daemon and listens on the unix socket.

Flags:
- `-sock` unix socket (defaults to `/tmp/krellin.sock`)
