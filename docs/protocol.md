# Protocol (v0)

Krellin clients communicate with the daemon using Actions and receive Events. All stateful mutations flow through Actions and are serialized by the Session Executor.

## Actions

Base fields: `action_id`, `session_id`, `agent_id`, `type`, `timestamp`, `payload`.

Action types:
- `run_command`
- `apply_patch`
- `freeze`
- `reset`
- `network_toggle`
- `containers_list`
- `revert`

## Events

Base fields: `event_id`, `session_id`, `timestamp`, `type`, `source`, `agent_id`, `payload`.

Event types:
- `session.started`
- `executor.busy`
- `executor.idle`
- `action.started`
- `action.finished`
- `terminal.output`
- `agent.message`
- `diff.ready`
- `freeze.created`
- `reset.completed`
- `network.changed`
- `containers.list`
- `error`

## Validation

- Unknown action/event types are rejected.
- Event `source` must be `system`, `executor`, or `agent`.
