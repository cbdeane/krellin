# Krellin

![Krellin TUI](docs/assets/krellin.png)

Krellin is a local-first coding runtime that pairs a daemon with per-repo Docker “capsules.” It gives LLM agents a controlled terminal + filesystem inside the capsule while keeping your host safe. Sessions are serialized, diffs are tracked, and you can reset or freeze environments.

**Status:** This project is under active development. Expect breaking changes and rough edges.

## What you get

- **Daemon + TUI**: `krellind` manages sessions; `krellin` is the terminal UI client.
- **Capsules**: one persistent container per repo; repo is mounted at `/workspace`.
- **Agent providers**: OpenAI, Anthropic, Gemini, Grok, LLaMA (OpenAI-compatible).
- **Tooling**: agent tool calls execute inside the capsule (shell, read/write, search, apply_patch).
- **Reset/Freeze**: reset a capsule to pinned image; freeze current state into a new image digest.
- **Safety defaults**: forbidden mounts, no docker socket, no host home mount.

## Quick start

```sh
scripts/run_tui.sh
```

This builds `krellind` + `krellin`, starts the daemon, and opens the TUI.

## Configuration (.krellinrc)

Krellin uses a repo-local `.krellinrc` (TOML). Image must be digest-pinned.

```toml
version = 1

[capsule]
image = "dokken/ubuntu-24.04@sha256:..."
user = "root" # optional; root skips cap-drop/no-new-privileges

[policy]
network = "on" # on | off

[resources]
cpus = 2
memory_mb = 4096

[freeze]
publish = ""
platforms = []
mode = "clean"
```

## Providers

Providers are stored in `~/.krellin/providers.json`. API keys are read from env vars.

```sh
krellin providers add --name myopenai --type openai --model gpt-4o-mini --api-key-env OPENAI_API_KEY
krellin providers add --name mygemini --type gemini --model gemini-2.0-flash --api-key-env GEMINI_API_KEY --base-url https://generativelanguage.googleapis.com/v1beta
```

## Common TUI commands

- `!<command>`: run a command inside the capsule
- `/agents`: manage providers
- `/diff`: show last diff
- `/reset`: recreate the capsule from the pinned image

## Docs

See `docs/README.md` for full docs:

- `docs/runbook.md` — quick start
- `docs/agents.md` — providers and agent tooling
- `docs/krellinrc.md` — config reference
- `docs/security.md` — safety model

## Notes

- This is a Docker-backed runtime; Docker must be installed and running.
- Reset recreates the capsule from the pinned digest; volumes are preserved unless disabled.
