# Daemon (v0)

Krellin runs a local daemon that manages sessions and streams events to clients.

## Transport

- Local-only socket (unix domain socket on supported platforms)
- Clients connect to subscribe to events and submit actions

## Responsibilities

- Owns all active sessions
- Manages client subscriptions
- Forwards Actions to per-session queues
- Emits Events from the session executor
