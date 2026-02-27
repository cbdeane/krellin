package images

import (
	"context"
	"testing"
)

type fakeRunner struct {
	out string
}

func (f fakeRunner) Run(ctx context.Context, args ...string) (string, error) {
	return f.out, nil
}

func TestResolveDigest(t *testing.T) {
	res := NewResolver(fakeRunner{out: "repo@sha256:abc"})
	digest, err := res.ResolveDigest(context.Background(), "image:tag")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if digest != "repo@sha256:abc" {
		t.Fatalf("unexpected digest: %q", digest)
	}
}
