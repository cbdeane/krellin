# Krellin Capsules: Local-First “Claude Code, but Safe” Runtime (v0 Plan)

## Goal

From inside a git repo:

- `cd repo && krellin`
- Krellin opens an agent session (VS Code extension preferred, TUI fallback)
- The agent has terminal takeover and can install packages, run tests, edit files
- Everything runs inside a Docker-backed **capsule** (one per repo)
- The user can always hit **Revert** to undo agent-caused repo/workspace changes
- The user can run **Freeze** to persist “system state” (installed packages/toolchains) as a pinned image for long-lived projects and sharing

This is **maximum power + instant undo**, not a “please be careful” guardrail product.

---

## Non-goals (v0)

- Perfect OS-level snapshot rollback across all filesystems/platforms
- Cloud execution
- Fine-grained network policy/allowlists (binary on/off is fine later)
- Multi-user, multi-tenant enterprise control plane
- Building our own VM runtime for macOS/Windows (we require Docker)

---

## Core Concepts

### Capsule
A per-repo persistent execution environment implemented as a Docker container:

- The capsule has its own filesystem state (toolchains, apt installs, configs)
- The capsule has a persistent home and cache volumes
- The repo is mounted into `/workspace` (bind mount)

**Key:** the agent terminal is always a PTY inside the capsule. No split-brain.

### `.krellinrc` (repo-local config)
A file in the git repo that pins the capsule environment and policy.
Checked into git so teams share identical setups.

### Revert (fast undo)
A user-facing “undo” that restores the repo/workspace to a last-known checkpoint.
In v0, we prioritize undoing repo changes (and optionally key dep dirs).

### Freeze (persist system state)
A user-driven “promote current capsule state into a new pinned image” operation:
- commits current capsule container -> new image digest
- optionally pushes to registry
- updates `.krellinrc` to pin that digest

This prevents “updating Krellin breaks my 6-month project”.

---

## UX Overview

### First run in a repo (instantiation)
If no `.krellinrc` exists, Krellin prompts once:

1) Use default Krellin image
2) Use existing image (tag/digest)
3) Create custom image (generate Dockerfile + build; optional publish)

Then it writes `.krellinrc` so future runs are zero-prompt.

### Normal run
`krellin`:
- ensures Docker is available
- reads `.krellinrc`
- ensures capsule exists (create or start)
- opens UI (VS Code extension preferred; TUI fallback)
- streams:
  - agent messages
  - terminal output
  - diff
  - timeline (checkpoints + freeze events)

### Safety feel
- No nag prompts by default
- Auto-checkpoints before risky steps (installs, bulk changes)
- Big obvious **Revert** button (timeline-based)

---

## Architecture Overview (Go + TS)

### Go Engine (runtime)
Responsible for:
- capsule lifecycle (create/start/stop)
- PTY terminal attach/stream
- executing commands (via PTY, v0)
- checkpoint + revert primitives
- freeze workflow (commit/tag/push + update `.krellinrc`)
- event stream to clients

### TS VS Code extension (thin UI)
- chat panel
- terminal panel (renders engine PTY stream)
- diff panel
- timeline + buttons (Revert, Freeze, Apply)

### CLI
- `krellin` (entrypoint)
- optionally `krellin init`, `krellin freeze`, `krellin revert`, etc.
Even if we support slash commands in chat, the CLI is the canonical interface.

---

## Docker Substrate Model

### Container
- One long-lived container per repo: `krellin-<repoId>`
- Runs idle command (`sleep infinity`) or tiny init
- Non-root user inside container
- No docker socket mount, no host home mount

### Mounts / Volumes
- Bind mount repo root -> `/workspace`
- Named volumes:
  - `krellin-<repoId>-home` -> `/home/dev`
  - `krellin-<repoId>-env`  -> `/env` (caches/toolchains)
- Goal: keep heavy churn (node_modules/venv) in container-managed locations if possible

### Security flags (baseline)
- drop caps: `--cap-drop=ALL`
- `--security-opt=no-new-privileges`
- resource limits (optional v0):
  - cpu/mem/pids

Network:
- v0 default: ON (agents need installs)
- later: OFF/allowlist policy

---

## Checkpoint + Revert (v0)

### Checkpoint creation
At minimum:
- record HEAD commit
- save dirty patch (if dirty) to `/env/checkpoints/<id>/workspace.patch`
- record file list changed (git diff --name-only)
- optionally: capture lockfiles or metadata

Auto-checkpoint triggers (suggested):
- before installs (`npm install`, `pip install`, `apt install`)
- before running repo scripts (package.json scripts)
- before applying multi-file patch

### Revert application
- restore repo to checkpoint state:
  - `git reset --hard <head>`
  - reapply patch if checkpoint captured dirty state and we want exact restore
- optional v0: wipe key dep dirs (node_modules/.venv) if we are storing them in container-local volumes

**Note:** “Revert” must be fast and reliable; perfection can come later.

---

## Freeze (persist system state)

### Purpose
Freeze locks the current capsule environment as a pinned base to prevent drift and share state.

### Operation
`krellin freeze`:
1) (optional) “clean freeze” step: remove obvious junk (tmp, build artifacts) based on heuristics/presets
2) `docker commit <container> <image-tag>`
3) resolve the image digest
4) if publish target configured:
   - push to registry
   - resolve pushed digest (multi-arch if applicable)
5) update `.krellinrc` to point to new digest
6) emit timeline event: `freeze.created`

### Local vs registry
- Solo dev: freeze can remain local (no registry required)
- Org: freeze pushes to registry (GHCR/ECR/etc.) for team reproducibility

---

## `.krellinrc` Proposal (TOML)

Example:

```toml
version = 1

[capsule]
image = "ghcr.io/krellin/capsules/debian-node20@sha256:..."
name  = "debian-node20"

[workspace]
path = "/workspace"
mount = "bind" # bind | sync (future)

[policy]
network = "on" # on | off (future allowlist)

[freeze]
publish = "" # optional registry repo, e.g. ghcr.io/acme/devcapsules/payments
platforms = ["linux/amd64", "linux/arm64"] # if publishing multi-arch
mode = "clean" # clean | as-is


Platform Notes (Docker required)
Linux

Best performance, bind mounts are fast

UID/GID mapping must allow write access

Later we can add a native backend, but v0 can stay Docker

macOS / Windows (Docker Desktop)

Still works as long as Docker Desktop runs

Bind mounts can be slow for huge repos / lots of small files

Strongly prefer putting heavy churn dirs into container volumes

Multi-arch images matter (mac often arm64)
