# Krellin v0 Modular Architecture

This document defines the idiomatic modularization for Krellin’s Go engine and supporting components.

The system is structured in **three layers**:

1. **Domain (pure logic, testable)**
2. **Services (orchestration + state)**
3. **Adapters (external systems: Docker, filesystem, UI clients)**

> Core rule: **policy and orchestration must not leak into substrate code**.

---

## Architectural Layers

### 1) Domain Layer
Pure logic, no external dependencies.

- Action/Event schemas
- Patch application logic
- Session state models
- Config schema

This layer should be:
- deterministic
- unit testable
- independent of Docker, filesystem, or UI

---

### 2) Service Layer
Coordinates system behavior.

- Session lifecycle
- Action queue + executor
- Capsule orchestration
- Policy enforcement

This is the “control plane” of Krellin.

---

### 3) Adapter Layer
Implements external integrations.

- Docker runtime
- PTY handling
- File system
- VS Code / TUI clients

Adapters should be:
- replaceable
- thin
- unaware of higher-level orchestration

---

## Top-Level Components

### Daemon (`daemon`)
- Owns all active sessions
- Hosts API + event stream
- Manages client connections (VS Code, TUI)
- Ensures single instance per machine

---

### Session (`session`)
- One session per repo
- Owns:
  - ActionQueue
  - SessionExecutor
  - Capsule handle
  - Client subscriptions

Implements:
- multi-agent Mode A (serialized execution)

---

### Session Executor (`executor`)
- Only component allowed to mutate state
- Pulls actions from queue
- Executes one action at a time

Responsible for:
- PTY execution
- patch application
- freeze/reset/network changes

---

### Action Queue (`queue`)
- FIFO queue per session
- Receives actions from agents or UI
- Feeds executor

---

### Capsule (`capsule`)
**Interface, not implementation**

Responsibilities:
- ensure capsule exists
- start/stop/reset container
- attach PTY
- commit image (freeze)
- set network mode
- query status

---

### Capsule Backend (`capsule/docker`)
- Docker-based implementation of capsule interface
- Handles:
  - containers
  - images
  - volumes

> Must remain a **dumb adapter**, no business logic.

---

### Patch (`patch`)
- Applies unified diffs atomically
- Tracks applied patches
- Supports revert (without git)

---

### Images (`images`)
- Handles:
  - freeze operations
  - image labeling
  - image listing
  - garbage collection

---

### Containers (`containers`)
- Tracks:
  - active capsules
  - volumes
  - container metadata

Powers `/containers` command.

---

### Config (`config`)
- Loads and validates `.krellinrc`
- Resolves image tags → digests
- Writes updated config (freeze)

---

### Policy (`policy`)
- Enforces safety rules:
  - forbidden mounts
  - resource limits
  - network defaults
  - unsafe mode gating

---

### Protocol (`protocol`)
- Defines:
  - Action schema
  - Event schema
- Handles:
  - serialization
  - versioning

---

### Repo (`repo`)
- Detects repo root
- Computes stable repo_id

---

### Store (`store`)
- Local daemon metadata:
  - session mapping
  - repo_id → capsule mapping

---

### PTY (`pty`)
- Handles terminal allocation and streaming

---

## Repository Layout (Go)
/cmd
  /krellin            # CLI entrypoint
  /krellind           # daemon entrypoint (optional; can be same binary)
/internal
  /app                # wiring: builds daemon with selected backends
  /daemon             # API server, auth (local), client connections
  /session            # session manager, session lifecycle
  /executor           # serialized action executor
  /queue              # action queue implementation
  /protocol           # action/event types + encoding + versioning
  /capsule            # capsule interfaces + domain types
    /docker           # docker backend implementation (adapter)
  /pty                # PTY management abstractions
  /patch              # apply_patch logic + bookkeeping
  /images             # freeze + image inventory + labels + GC policies
  /containers         # container/volume inventory; repo_id mapping
  /config             # .krellinrc parsing/writing + validation
  /policy             # safety defaults, forbidden mounts, unsafe gating
  /repo               # repo root detection + repo_id derivation
  /store              # local metadata store (bolt/sqlite/json) for daemon
/pkg (optional)
  /client             # reusable client lib for TUI / tests
/vscode
  ...                 # TS extension
/docs


---

## Key Interfaces

### Capsule Interface

```go
type Capsule interface {
  Ensure(repoID string, cfg Config) (Handle, error)
  Start(handle Handle) error
  Stop(handle Handle) error
  Reset(handle Handle, imageDigest string) error
  AttachPTY(handle Handle) (PTYConn, error)
  Commit(handle Handle, opts CommitOptions) (string, error)
  SetNetwork(handle Handle, enabled bool) error
  Status(handle Handle) (CapsuleStatus, error)
}

### Executor Interface

type Executor interface {
  Submit(action Action)
  Run(ctx context.Context)
}

### Event Emitter

type Emitter interface {
  Emit(event Event)
}

## Dependency Rules (IMPORTANT)

Allowed Dependencies

daemon → session

session → executor, queue, capsule, patch

executor → capsule, patch, protocol

capsule/docker → external Docker APIs only

config → used by session/daemon

## Forbidden Dependencies

capsule/docker must NOT import:

session

executor

policy

daemon must NOT implement business logic

protocol must NOT depend on runtime code

## Client Architecture
VS Code Extension
/vscode
  /src
    /client
    /ui
    /terminal
    /diff
    /state

### Responsibilities:

- connect to daemon
- render UI
- forward user input
- display events

## Core Design Principle

All system behavior flows through Actions → Executor → Events

- Clients send Actions
- Executor mutates state
- Daemon emits Events

This guarantees:

- determinism
- observability
- replayability
- extensibility

## Future-Proofing

This modularization allows:

- swapping Docker backend → native Linux backend
- adding new clients (CLI, web, etc.)
- extending multi-agent capabilities
- adding action prioritization / cancellation
- implementing audit logs or replay

## Summary

This structure ensures:

- clean separation of concerns
- deterministic execution
- safe extensibility
- minimal coupling between layers

Krellin becomes a local control plane for agent execution, not just a tool.
