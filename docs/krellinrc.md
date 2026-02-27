# .krellinrc Reference (v0)

Krellin uses a repo-local `.krellinrc` file as the source of truth for capsule configuration.

## Schema (TOML)

```toml
version = 1

[capsule]
# Must be a digest-pinned image reference.
image = "ghcr.io/krellin/capsules/debian@sha256:..."

[policy]
network = "on" # on | off

[resources]
cpus = 2
memory_mb = 4096

[freeze]
publish = "" # optional registry path
platforms = ["linux/amd64", "linux/arm64"]
mode = "clean" # clean | as-is
```

## Rules

- `capsule.image` must be pinned to a digest (`@sha256:...`).
- If a user supplies a tag, higher-level code resolves it to a digest before writing.
- Defaults: `policy.network=on`, `resources.cpus=2`, `resources.memory_mb=4096`, `freeze.mode=clean`.

## Default Instantiation

If `.krellinrc` is missing, the daemon initializes it using `ubuntu:latest` and resolves it to a digest before writing.
