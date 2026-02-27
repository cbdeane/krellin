package session

import (
	"context"
	"testing"

	"krellin/internal/capsule"
	"krellin/internal/policy"
)

type fakeCapsule struct {
	called bool
}

func (f *fakeCapsule) Ensure(ctx context.Context, cfg capsule.Config) (capsule.Handle, error) {
	f.called = true
	return capsule.Handle{}, nil
}

func (f *fakeCapsule) Start(ctx context.Context, handle capsule.Handle) error { return nil }
func (f *fakeCapsule) Stop(ctx context.Context, handle capsule.Handle) error { return nil }
func (f *fakeCapsule) Reset(ctx context.Context, handle capsule.Handle, imageDigest string, preserveVolumes bool) error {
	return nil
}
func (f *fakeCapsule) AttachPTY(ctx context.Context, handle capsule.Handle) (capsule.PTYConn, error) {
	return nil, nil
}
func (f *fakeCapsule) Commit(ctx context.Context, handle capsule.Handle, opts capsule.CommitOptions) (string, error) {
	return "", nil
}
func (f *fakeCapsule) SetNetwork(ctx context.Context, handle capsule.Handle, enabled bool) error { return nil }
func (f *fakeCapsule) Status(ctx context.Context, handle capsule.Handle) (capsule.Status, error) { return capsule.Status{}, nil }

func TestCapsuleManagerValidatesMounts(t *testing.T) {
	caps := &fakeCapsule{}
	mgr := CapsuleManager{
		Capsule: caps,
		Policy:  policy.DefaultPolicy("/repo", "/home/user"),
	}

	_, err := mgr.Ensure(context.Background(), capsule.Config{RepoRoot: "/etc", RepoID: "repo1"})
	if err == nil {
		t.Fatalf("expected mount validation error")
	}
	if caps.called {
		t.Fatalf("capsule ensure should not be called on invalid mount")
	}
}

func TestCapsuleManagerAllowsRepoRoot(t *testing.T) {
	caps := &fakeCapsule{}
	mgr := CapsuleManager{
		Capsule: caps,
		Policy:  policy.DefaultPolicy("/repo", "/home/user"),
	}

	_, err := mgr.Ensure(context.Background(), capsule.Config{RepoRoot: "/repo", RepoID: "repo1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !caps.called {
		t.Fatalf("expected capsule ensure to be called")
	}
}
