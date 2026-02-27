# Daemon Transport (v0)

The daemon speaks line-delimited JSON over a local unix socket.

## Handshake

Clients must send a `connect` message as the first JSON line:

```json
{"type": "connect", "session_id": "<session-id>"}
```

The daemon responds with:

```json
{"type": "connected", "session_id": "<session-id>"}
```

## Messages

- Actions are sent by clients to the daemon as JSON lines.
- Events are streamed back to clients as JSON lines.
