# Krellin v0 Architecture: Capsules, Freeze, Safety, and Daemon Model

## Product Goal

From inside a repo:

- `cd repo && krellin`
- Krellin opens an agent session that feels like Claude Code / OpenCode
- The agent has terminal takeover and can install tools, run tests, and edit files
- All execution happens inside a **persistent per-repo capsule** (Docker-backed)
- The host machine stays clean and safe by default
- The user can run **Freeze** to pin system state (installed packages/toolchains) for long-lived projects

Krellin is about **protecting the host computer** from agent chaos, not about managing the user’s git workflow.

---

## Non-Goals (v0)

- Agent-managed git operations (no auto-commits, no auto-reset, no "smart git")
- Perfect filesystem snapshot rollback on all platforms
- Cloud execution
- Fine-grained egress logging or allowlists
- Ad-hoc automation for non-idiomatic repo conventions

---

## Key Decisions (v0)

### 1) Daemon Model
- A **single daemon** manages **multiple concurrent repo sessions**.
- Users may run multiple projects simultaneously; Krellin must not bottleneck them.

### 2) Capsule Persistence
- Each repo has a **persistent capsule container**.
- We do **not** delete/recreate containers “willy nilly.”
- Capsules can be stopped when idle and restarted when needed.
- Capsule recreation from pinned image happens only on:
  - explicit user **Reset**, or
  - failure fallback when the capsule becomes unrecoverable.

### 3) Revert Semantics
- v0 revert is **repo-only** and does **not** involve agent-driven git operations.
- Krellin does not inject prompts telling the agent to manipulate git.
- Users manage their repos themselves, including dirty working trees.

### 4) Freeze Semantics
- Freeze creates a new image from the current capsule state and pins it to `.krellinrc`.
- This prevents “updating Krellin changes the environment for active projects.”

### 5) Cache/Toolchain Persistence
- Persistence happens **inside the capsule** via Docker volumes and/or container state.
- We do **not** persist language caches onto the host machine in v0.

### 6) Network Policy
- Default network is **ON**.
- User can toggle via `/network` (saved to `.krellinrc`).

### 7) Resource Defaults
- Default CPU: **2**
- Default memory: **4 GB**
- Exposed via `.krellinrc` and surfaced via `/containers`.

### 8) Safety Policy
- Hard-forbidden mounts by default (host `$HOME`, docker socket, root filesystem).
- If users want to open it up, they must opt into `/unsafe` with big warnings.

### 9) Clean Freeze
- v0 “clean freeze” does **not** bake in the user’s home dir.
- We may reassess later.
- Clean freeze clears temp dirs (e.g., `/tmp`) and other obvious junk.

---

## Primary Abstractions

### Capsule
A per-repo persistent execution environment:

- Implemented as a Docker container: `krellin-<repoId>`
- Repo is mounted at `/workspace`
- Persistent volumes:
  - `krellin-<repoId>-home` mounted at `/home/dev` (persisted user home)
  - `krellin-<repoId>-env` mounted at `/env` (persisted caches/tooling)
- The agent terminal is a **PTY inside the capsule** (no split brain)

### `.krellinrc`
Repo-local config file, intended to be checked into git so teams share a consistent capsule.

Minimum v0 fields:
- pinned image digest
- network mode
- resource defaults
- publish target (optional)

---

## `.krellinrc` (TOML) Schema v0

```toml
version = 1

[capsule]
# Pinned immutable truth; always store digest
image = "ghcr.io/krellin/capsules/debian@sha256:..."

[policy]
network = "on"  # on | off

[resources]
cpus = 2
memory_mb = 4096

[freeze]
# Optional: where to push freeze images (org registry)
publish = ""  # e.g. ghcr.io/acme/devcapsules/payments

# Optional: freeze behavior
mode = "clean" # clean | as-is

Rules:

If user supplies an image tag, Krellin resolves it to a digest and writes the digest.

.krellinrc is the source of truth for capsule image selection.

Lifecycle
A) Project Instantiation (first run)

If .krellinrc is missing:

Prompt user once:

Use default Krellin image

Use existing image (tag or digest)

Create custom image (generate Dockerfile + build; optional publish)

Then:

resolve and pin digest into .krellinrc

create capsule container from pinned digest

start session + UI

B) Normal Run

krellin:

ensures daemon is running (VS Code extension should also be able to start it)

reads .krellinrc

ensures capsule exists:

create if missing

start if stopped

attaches a PTY shell inside the capsule

UI connects to daemon and streams:

agent messages

terminal output

diffs

timeline events

C) Reset (explicit)

/reset (or CLI equivalent):

stops and removes the capsule container

recreates it from the pinned digest in .krellinrc

keeps volumes policy-defined (v0: decide whether to preserve home/env volumes or recreate them; default should preserve unless user asks for full wipe)

D) Failure Fallback

If capsule/session becomes unrecoverable:

create a new capsule from pinned image

surface what happened clearly in UI

do not silently destroy data without warning

Revert (Repo-only) - v0

Revert is not “git restore.” It is “discard the agent’s applied patch.”

Behavior:

Krellin shows diffs from agent edits

User chooses Apply or Reject

If user hits Revert:

Krellin restores repo state through its own patch application bookkeeping

Krellin does not execute arbitrary git commands automatically

Notes:

Dirty working trees are allowed.

If a patch cannot be cleanly reverted, we fail safe and offer Reset from pinned image.

(Implementation details intentionally omitted here.)

Freeze - v0
Purpose

Persist package/toolchain/system changes so projects don’t drift with Krellin updates.

Behavior

/freeze:

creates a new image based on the current capsule state

pins the resulting digest into .krellinrc

optionally pushes to a registry if configured

Clean Freeze v0

clear /tmp and other obvious temp dirs

do not bake user home into the image (home remains a volume)

Image Versioning + Visibility

We will provide /containers that shows:

current pinned image digest

list of freeze images for this repo (timestamp, size, digest)

disk usage summary

cleanup options:

delete freezes older than X days

keep last N freezes

delete all non-pinned freezes

By default, the originally pinned base image remains untouched.

All images/containers/volumes created by Krellin are labeled with:

krellin.repo_id

krellin.repo_root

krellin.kind = capsule|freeze|base

krellin.created_at

Safety Defaults

Hard forbidden by default:

mounting host $HOME or arbitrary host paths outside repo root

mounting /var/run/docker.sock

privileged containers

arbitrary capability adds

Allowed by default:

bind mount repo root only

Escape hatch:

/unsafe enables additional mounts/capabilities with explicit warnings and requires explicit configuration.

Commands (Conceptual Surface)

krellin : open/attach session

/network on|off : toggle capsule network mode (persisted)

/freeze : commit capsule -> new image and pin digest

/containers : list capsule + images + cleanup controls

/reset : recreate capsule from pinned digest

/unsafe : enable dangerous capabilities/mounts (explicit)

Platform Notes

We standardize methodology by requiring Docker:

Linux: Docker Engine

macOS/Windows: Docker Desktop

Methodology does not change; only substrate performance differs.
We prioritize a consistent capsule abstraction and event-driven UX across platforms.

Multi-Agent Support (v0)

A session may have N agents connected.

Agents may propose plans, patches, and command sequences concurrently.

A Session Executor serializes stateful operations:

PTY I/O

file patch application

freeze/reset/network toggles

The UI shows agent identity on messages and proposals.

Conflicting patch proposals must be resolved explicitly (UI or arbiter).

That’s enough to say “yes, supports multi-agent.”


## Multi-Agent Execution Model (v0)

Krellin supports multiple agents within a single session using a **serialized execution model**.

### Model Overview

- A session may have **multiple agents** (planner, reviewer, executor, etc.)
- Agents may operate concurrently at the reasoning level
- All **stateful operations are serialized** through a single execution path

This ensures deterministic behavior and prevents race conditions.

---

## Session Executor

Each session owns a **Session Executor**, which is the only component allowed to mutate state.

The Session Executor is responsible for:

- exclusive access to the capsule PTY
- executing commands inside the capsule
- applying file patches to the workspace
- handling system operations (freeze, reset, network changes)
- enforcing strict operation ordering

> The Session Executor acts as a single-threaded control plane for the session.

---

## Action Queue

All agent actions are routed through a **per-session Action Queue**.

### Flow
Agent → Action → Queue → Executor → Capsule


### Action Types (v0)

- `run_command`
- `apply_patch`
- `freeze`
- `reset`
- `network_toggle`

(Checkpointing may be internal and not exposed as a first-class action.)

### Properties

- FIFO ordering (v0)
- Each action includes:
  - `action_id`
  - `agent_id`
  - `timestamp`
- The executor processes **one action at a time**

This guarantees deterministic execution and enables a clean timeline.

---

## Agent Identity

All actions and events must include agent attribution:

- `agent_id`
- `agent_name` (optional)
- `role` (planner, executor, reviewer, etc.)

### Implications

- UI can attribute messages, diffs, and commands to specific agents
- Improves debuggability and user trust
- Enables future role-based behavior

---

## Execution Locking

The Session Executor enforces a single rule:

> Only one action may execute at a time.

During execution:
- the executor holds an exclusive lock
- no other agent may mutate state

Optional event signals:

- `executor.busy`
- `executor.idle`
- `action.started`
- `action.finished`

---

## Patch-Based File Mutation

Agents do not directly modify files.

All file changes must go through:

- `apply_patch` actions

### Rationale

- Enables deterministic multi-agent collaboration
- Avoids uncontrolled file mutations
- Makes revert behavior reliable
- Keeps all state transitions observable

---

## Enforcement Rule

> No agent may bypass the Session Executor.

All operations that interact with the capsule or workspace must pass through:

- the Action Queue
- the Session Executor

This includes:
- command execution
- file modification
- system state changes

---

## Design Rationale

This model provides:

- deterministic execution
- protection against race conditions
- clear observability via action timeline
- clean multi-agent coordination without complexity

---

## Out of Scope (v0)

The following are intentionally not supported:

- parallel state mutation by multiple agents
- DAG-based or graph execution models
- CRDT/OT-based collaborative editing
- agent-driven git operations

These may be considered in future iterations.

## Action Schema (v0)

Actions represent all stateful operations within a session.  
They are produced by agents (or the system) and executed exclusively by the Session Executor.

---

### Base Action Structure

```json
{
  "action_id": "uuid",
  "session_id": "uuid",
  "agent_id": "string",
  "type": "string",
  "timestamp": "iso8601",
  "payload": {}
}

Fields

- action_id — unique identifier for the action
- session_id — owning session
- agent_id — originator of the action
- type — action type (see below)
- timestamp — creation time
- payload — action-specific data

Action Types
1) run_command

Execute a command inside the capsule PTY.

{
  "type": "run_command",
  "payload": {
    "command": "string",
    "cwd": "/workspace",
    "env": {}
  }
}

Notes:

Commands are executed inside the capsule shell

Output is streamed via terminal events

2) apply_patch

Apply a unified diff patch to the workspace.

{
  "type": "apply_patch",
  "payload": {
    "patch": "unified_diff_string"
  }
}

Notes:

This is the only allowed file mutation mechanism

Patch application must be atomic (success or fail)

3) freeze

Persist the current capsule state into a new image and update .krellinrc.

{
  "type": "freeze",
  "payload": {
    "mode": "clean" 
  }
}

Notes:

mode can be clean or as-is

Clean mode removes temp artifacts before commit

4) reset

Recreate the capsule from the pinned image.

{
  "type": "reset",
  "payload": {
    "preserve_volumes": true
  }
}

Notes:

Used for explicit reset or failure recovery

5) network_toggle

Enable or disable network access for the capsule.

{
  "type": "network_toggle",
  "payload": {
    "enabled": true
  }
}

Event Schema (v0)

Events are emitted by the daemon and streamed to clients (VS Code, TUI).

They represent state changes, execution progress, and outputs.

Base Event Structure

{
  "event_id": "uuid",
  "session_id": "uuid",
  "timestamp": "iso8601",
  "type": "string",
  "source": "system | executor | agent",
  "agent_id": "string | null",
  "payload": {}
}

Fields

event_id — unique identifier

session_id — owning session

timestamp — event time

type — event type (see below)

source — origin of event

agent_id — optional attribution

payload — event-specific data

Event Types
1) session.started

{
  "type": "session.started",
  "payload": {
    "repo_root": "/path/to/repo",
    "capsule_name": "krellin-abc123"
  }
}


2) executor.busy / executor.idle
{
  "type": "executor.busy",
  "payload": {
    "action_id": "uuid"
  }
}

{
  "type": "executor.idle",
  "payload": {}
}

3) action.started / action.finished
{
  "type": "action.started",
  "payload": {
    "action_id": "uuid",
    "type": "run_command"
  }
}

{
  "type": "action.finished",
  "payload": {
    "action_id": "uuid",
    "status": "success | failure",
    "error": null
  }
}

4) terminal.output
{
  "type": "terminal.output",
  "payload": {
    "stream": "stdout | stderr",
    "data": "string"
  }
}

Notes:

Raw terminal stream from PTY

Should be chunked and streamed incrementally

5) agent.message
{
  "type": "agent.message",
  "payload": {
    "content": "markdown_string"
  }
}

6) diff.ready
{
  "type": "diff.ready",
  "payload": {
    "patch": "unified_diff_string",
    "files": ["file1", "file2"]
  }
}

7) freeze.created
{
  "type": "freeze.created",
  "payload": {
    "image": "ghcr.io/...@sha256:...",
    "size_bytes": 123456789
  }
}

8) reset.completed
{
  "type": "reset.completed",
  "payload": {}
}

9) network.changed
{
  "type": "network.changed",
  "payload": {
    "enabled": true
  }
}

10) error
{
  "type": "error",
  "payload": {
    "message": "string",
    "action_id": "uuid | null"
  }
}

Design Guarantees

- All state mutations originate from Actions
- All state changes are observable via Events
- Execution is strictly serialized by the Session Executor
- Agent attribution is preserved across all actions and events
- The event stream is sufficient to reconstruct session behavior

## Session Lifecycle Diagram (v0)

### 1) Session + Capsule State Machine (Mermaid)

```mermaid
stateDiagram-v2
  [*] --> NoSession

  NoSession --> EnsureDaemon : krellin / VS Code attach
  EnsureDaemon --> ResolveRepo : daemon running

  ResolveRepo --> NoConfig : .krellinrc missing
  ResolveRepo --> HaveConfig : .krellinrc present

  NoConfig --> Instantiate : user selects image (default/existing/custom)
  Instantiate --> HaveConfig : .krellinrc written (digest pinned)

  HaveConfig --> EnsureCapsule : ensure capsule container exists
  EnsureCapsule --> CreateCapsule : capsule missing
  EnsureCapsule --> StartCapsule : capsule exists but stopped
  EnsureCapsule --> AttachCapsule : capsule running

  CreateCapsule --> AttachCapsule : container created + started
  StartCapsule --> AttachCapsule : container started

  AttachCapsule --> SessionActive : PTY attached, UI subscribed

  SessionActive --> Idle : no actions running
  Idle --> Busy : ActionQueue non-empty
  Busy --> Idle : Action finished

  Idle --> Freeze : user/agent requests freeze
  Freeze --> Idle : image committed (+ optional push), .krellinrc updated

  Idle --> Reset : user requests reset
  Reset --> AttachCapsule : capsule recreated from pinned image

  Busy --> Failure : unrecoverable exec error / corruption
  Failure --> Reset : fallback reset from pinned image
  Reset --> AttachCapsule : recovered

  SessionActive --> SessionClosed : UI disconnect & no clients
  SessionClosed --> CapsuleStopped : stop capsule after idle timeout (optional)
  CapsuleStopped --> [*]


---

```md
### 2) End-to-End Sequence (Mermaid)

```mermaid
sequenceDiagram
  autonumber
  participant U as User
  participant V as VS Code Extension
  participant C as CLI (krellin)
  participant D as Daemon
  participant X as Session Executor
  participant Q as Action Queue
  participant K as Capsule (Docker container)
  participant R as Registry (optional)

  U->>C: run `krellin` in repo
  C->>D: ensure daemon running
  D-->>C: daemon ready

  C->>D: resolve repo root + repo_id
  D->>D: read .krellinrc
  alt .krellinrc missing
    D-->>C: prompt instantiation choices
    U->>C: choose default/existing/custom image
    C->>D: selection + (optional) build/publish config
    D->>D: resolve image tag -> digest
    D->>D: write .krellinrc (digest pinned)
  end

  C->>D: start/attach session(repo_id)
  D->>D: ensure capsule container exists
  alt capsule missing
    D->>K: docker create+start (pinned digest)
  else capsule stopped
    D->>K: docker start
  else capsule running
    D->>K: no-op
  end

  Note over D,K: Capsule is persistent per repo; not recreated unless reset/failure.

  C->>V: launch/open VS Code (if configured)
  V->>D: connect to event stream
  D-->>V: session.started

  D->>K: attach PTY (interactive shell)
  D-->>V: terminal.output (banner / prompt)
  Note over V: "first terminal output" achieved

  rect rgb(240,240,240)
  Note over U,V: Normal multi-agent operation (Mode A)
  U->>V: chat prompt(s)
  V->>D: agent produces action(s)
  D->>Q: enqueue Action(s) with agent_id
  Q->>X: dequeue next action
  X->>K: execute (PTY / apply_patch / etc.)
  X-->>D: action.started/action.finished
  D-->>V: terminal.output / agent.message / diff.ready
  end

  alt User runs Freeze
    U->>V: /freeze
    V->>D: enqueue freeze action
    Q->>X: freeze
    X->>K: clean temp (optional)
    X->>K: docker commit -> new image
    alt publish configured
      X->>R: docker push
      R-->>X: pushed digest
    end
    X->>D: update .krellinrc pinned digest
    D-->>V: freeze.created
  end

  alt Failure fallback
    X-->>D: error (unrecoverable)
    D-->>V: error
    D->>K: stop/remove capsule container
    D->>K: recreate capsule from pinned digest
    D-->>V: reset.completed
    D->>K: reattach PTY
    D-->>V: terminal.output (ready)
  end

  alt User requests Reset
    U->>V: /reset
    V->>D: enqueue reset action
    Q->>X: reset
    X->>K: stop/remove capsule container
    X->>K: recreate from pinned digest
    X-->>D: reset.completed
    D-->>V: reset.completed
  end

## Image Lifecycle Diagram (v0)

### Image + Capsule Evolution (Mermaid)

```mermaid
flowchart TD
  A[Project Instantiation] --> B{.krellinrc exists?}

  B -- No --> C[Select Base Image<br/>default | existing | custom]
  C --> D[Resolve to Digest<br/>tag -> sha256]
  D --> E[Write .krellinrc<br/>capsule.image = base@sha256]

  B -- Yes --> E

  E --> F[Create/Start Capsule Container<br/>krellin-<repoId>]
  F --> G[Capsule Running<br/>State changes accumulate]
  G --> H{Freeze?}

  H -- No --> G

  H -- Yes --> I[Clean Freeze (v0)<br/>wipe /tmp, /var/tmp<br/>do NOT bake home]
  I --> J[docker commit<br/>container -> new image]
  J --> K{Publish configured?}

  K -- No --> L[Local Image Only<br/>new@sha256 exists locally]
  K -- Yes --> M[Push to Registry<br/>docker push]
  M --> N[Resolve Published Digest<br/>multi-arch if applicable]

  L --> O[Pin New Digest in .krellinrc<br/>capsule.image = new@sha256]
  N --> O

  O --> P[Future Runs Use Pinned Digest]
  P --> Q{Reset?}

  Q -- No --> F

  Q -- Yes --> R[Reset: Recreate Capsule<br/>from pinned digest]
  R --> F

  %% Cleanup / GC
  O --> S[/containers: list images]
  S --> T{Cleanup policy}
  T --> U[Keep pinned base + current pinned]
  T --> V[Delete freezes older than X]
  T --> W[Keep last N freezes]
  T --> X[Prune dangling/unreferenced]

  %% Invariants
  subgraph Invariants
    Y[Base image digest is immutable]
    Z[Freeze produces new digest]
    AA[.krellinrc always stores digest]
  end

Image Lifecycle Invariants (v0)

Digests only: .krellinrc must store image@sha256:... (tags are resolved, never authoritative).

Base is immutable: The originally selected base digest is never modified.

Freeze creates a new image: freeze always produces a new digest and updates .krellinrc to pin it.

Reset uses pinned digest: Reset recreates the capsule container from the currently pinned digest.

GC never breaks projects by default: cleanup must never remove the currently pinned digest or the originally pinned base digest unless the user explicitly forces it.

Recommended Labels for Tracking

All images created by Krellin should be labeled:

krellin.repo_id

krellin.repo_root

krellin.kind = base|freeze

krellin.created_at

krellin.pinned = true|false (optional convenience)

This enables /containers to reliably list and clean images per project.
