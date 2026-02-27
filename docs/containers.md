# Containers & Labels (v0)

Krellin labels every capsule container to support inventory and cleanup.

## Labels (capsules)

- `krellin.repo_id`
- `krellin.repo_root`
- `krellin.kind=capsule`
- `krellin.created_at` (optional)

These labels are used by `/containers` to list and filter resources.

## Image Inventory (v0)

Krellin also lists images labeled with `krellin.kind` for freeze/base tracking.

## Cleanup Policies (v0)

Planned policies:
- Keep last N freezes
- Delete freezes older than X
- Delete unpinned freezes
