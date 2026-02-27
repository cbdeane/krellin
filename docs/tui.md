# TUI (v0)

The TUI connects to the daemon and renders:
- Timeline
- Output (terminal stream)
- Diff output

Input is typed into the bottom input box:
- `command` or `!command` sends a `run_command` action to the capsule.
- `/agents` lists configured agent backends.
