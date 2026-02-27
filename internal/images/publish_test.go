package images

import (
	"context"
	"testing"
)

type recordRunner struct {
	calls [][]string
}

func (r *recordRunner) Run(ctx context.Context, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	return "", nil
}

func TestPushTagsAndPushes(t *testing.T) {
	runner := &recordRunner{}
	pub := NewPublisher(runner)
	_, err := pub.Push(context.Background(), "img:local", "ghcr.io/acme/app:latest", []string{"linux/amd64", "linux/arm64"})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 docker calls, got %d", len(runner.calls))
	}
	if runner.calls[0][1] != "tag" {
		t.Fatalf("expected tag, got %v", runner.calls[0])
	}
	if runner.calls[1][1] != "push" {
		t.Fatalf("expected push, got %v", runner.calls[1])
	}
}
