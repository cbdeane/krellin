# Security Model (v0)

Krellin is local-first and assumes untrusted client input. Safety defaults are enforced in the daemon and capsule backend.

## Defaults

- Network: on by default
- Resources: 2 CPU, 4 GB
- Container: non-privileged, no extra capabilities, no docker socket mount
- Mounts: only the repo root bind-mounted to `/workspace`

## Forbidden Mounts (default)

- Host `$HOME`
- `/var/run/docker.sock`
- `/` and arbitrary host paths outside the repo root

## /unsafe

`/unsafe` explicitly relaxes mount constraints. It must be opt-in and loud. When unsafe is enabled, additional mounts may be allowed, but the user assumes the risk.

## Audit Checklist (v0)

- [x] No privileged containers
- [x] Docker socket mount forbidden by default
- [x] Host home mount forbidden by default
- [x] Default network ON; configurable
- [x] Resource defaults enforced (2 CPU, 4 GB)
- [x] All stateful operations serialized via executor
