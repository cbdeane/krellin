package containers

import (
	"context"
	"testing"
	"time"
)

type recordingRunner struct {
	outputs map[string]string
	calls   [][]string
}

func (r *recordingRunner) Run(ctx context.Context, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	key := joinArgs(args)
	if r.outputs == nil {
		return "", nil
	}
	return r.outputs[key], nil
}

func TestCleanupPolicyValidation(t *testing.T) {
	inv := New(&fakeRunner{})
	if err := inv.Cleanup(context.Background(), CleanupPolicy{KeepLastN: -1}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestCleanupDeletesOldUnpinned(t *testing.T) {
	runner := &recordingRunner{outputs: map[string]string{
		"docker images --filter label=krellin.kind --format {{.ID}}|{{.Repository}}|{{.Tag}}": "img1|repo|tag\nimg2|repo|tag",
		"docker inspect -f {{json .Config.Labels}} img1":                                            "{\"krellin.created_at\":\"2026-01-01T00:00:00Z\",\"krellin.pinned\":\"false\"}",
		"docker inspect -f {{json .Config.Labels}} img2":                                            "{\"krellin.created_at\":\"2026-02-26T00:00:00Z\",\"krellin.pinned\":\"true\"}",
	}}
	inv := NewWithClock(runner, func() time.Time { return time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC) })
	err := inv.Cleanup(context.Background(), CleanupPolicy{DeleteOlderThan: "24h", DeleteUnpinned: true})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if len(runner.calls) == 0 {
		t.Fatalf("expected docker rmi call")
	}
	last := runner.calls[len(runner.calls)-1]
	if len(last) < 3 || last[1] != "rmi" || last[2] != "img1" {
		t.Fatalf("unexpected rmi call: %v", last)
	}
}
