# Krellin

![Krellin TUI](docs/assets/krellin.png)

Krellin is a local-first runtime for AI-assisted development. It pairs a daemon with per‑repo Docker capsules so agents can run real commands and edit real files without touching your host machine. It serializes execution, makes every change traceable, and provides deterministic resets.
Krellin is a local control plane for executing AI agents inside isolated runtime capsules.
Krellin gives agents full control inside the capsule, while enforcing a hard boundary that prevents any modification of the host system.

**Status:** Under active development. Expect breaking changes and rough edges.

## Why it exists

Krellin is built so you can let an agent do real work without worrying about your host machine. The core idea is simple: **do everything in a sandboxed capsule**, and make that sandbox **repeatable and resettable**.

- **Safety by default**: no host home mount, no Docker socket, no silent privilege escalation.
- **Deterministic state**: capsules are pinned to immutable image digests, not floating tags.
- **Auditable changes**: everything flows through a serialized action queue and executor with diffs and logs.
- **Fast rollback**: reset the capsule to a known image, or freeze current state for the team.

## What you get

- **Daemon + TUI**: `krellind` manages sessions; `krellin` is the terminal UI client.
- **Capsules**: one persistent container per repo; repo is mounted at `/workspace`.
- **Agent providers**: Gemini and LLaMA (tested). OpenAI, Anthropic, and Grok supported via adapters.
- **Tooling**: agent tool calls run inside the capsule (shell, read/write, search, apply_patch).
- **Reset/Freeze**: reset to a pinned image; freeze the current state into a new digest.
- **Safety defaults**: forbidden mounts, no Docker socket, no host home mount.

## Quick start

```sh
curl -fsSL https://raw.githubusercontent.com/cbdeane/krellin/main/scripts/quickstart.sh | bash
```

or, if you already have the repo:

```sh
scripts/run_tui.sh
```

This builds `krellind` + `krellin`, starts the daemon, and opens the TUI.

## Configuration (.krellinrc)

Krellin uses a repo-local `.krellinrc` (TOML). Image must be digest-pinned. See `.krellinrc.example` to get started.

```toml
version = 1

[capsule]
image = "dokken/ubuntu-24.04@sha256:..."
# Run as root inside the capsule (default for development ergonomics).
# This allows unrestricted package installs and system changes *within the container*.
# Host system remains isolated and protected by default.
user = "root"

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

## What this is not

- Not a git automation tool. You keep control of your repo history.
- Not a cloud service. Everything runs locally.

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

## License

Krellin is licensed under the Business Source License 1.1. Commercial licensing terms are available for hosted use, redistribution, or embedding.

- `LICENSE-BSL.txt`
- `LICENSE-COMMERCIAL.md`
