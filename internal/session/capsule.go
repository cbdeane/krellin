package session

import (
	"context"

	"krellin/internal/capsule"
	"krellin/internal/policy"
)

// CapsuleManager enforces policy before calling the capsule backend.
type CapsuleManager struct {
	Capsule capsule.Capsule
	Policy  policy.Policy
}

func (m CapsuleManager) Ensure(ctx context.Context, cfg capsule.Config) (capsule.Handle, error) {
	// Only validate the repo root mount; HOME is set inside container, not mounted.
	if err := policy.ValidateMount(m.Policy, cfg.RepoRoot); err != nil {
		return capsule.Handle{}, err
	}
	return m.Capsule.Ensure(ctx, cfg)
}
