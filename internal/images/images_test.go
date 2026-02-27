package images

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	calls int
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) (string, error) {
	f.calls++
	// First inspect fails, then pull succeeds, then inspect returns digest.
	if f.calls == 1 && args[1] == "inspect" {
		return "", errors.New("inspect failed")
	}
	if args[1] == "pull" {
		return "pulled", nil
	}
	return "repo@sha256:abc", nil
}

func TestResolveDigest(t *testing.T) {
	res := NewResolver(&fakeRunner{})
	digest, err := res.ResolveDigest(context.Background(), "image:tag")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if digest != "repo@sha256:abc" {
		t.Fatalf("unexpected digest: %q", digest)
	}
}
