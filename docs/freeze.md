# Freeze (v0)

Freeze commits the current capsule container into a new image and pins the digest into `.krellinrc`.

## Flow

1) Commit container -> image
2) Resolve digest
3) (Optional) publish to registry and resolve published digest
4) Write digest to `.krellinrc`
5) Emit `freeze.created`
