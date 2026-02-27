package docker

import (
	"context"
	"strings"
	"testing"

	"krellin/internal/capsule"
)

type call struct {
	args []string
}

type fakeRunner struct {
	calls []call
	outputs map[string]string
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) (string, error) {
	f.calls = append(f.calls, call{args: args})
	if f.outputs == nil {
		return "", nil
	}
	key := joinArgs(args)
	return f.outputs[key], nil
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func TestEnsureCreatesWhenMissing(t *testing.T) {
	fr := &fakeRunner{
		outputs: map[string]string{
			"docker ps -a --filter name=krellin-repo1 --format {{.ID}}": "",
		},
	}
	c := New(fr)
	_, err := c.Ensure(context.Background(), capsule.Config{
		RepoID:      "repo1",
		RepoRoot:    "/repo",
		ImageDigest: "img@sha256:abc",
		NetworkOn:   true,
		CPUs:        2,
		MemoryMB:    4096,
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	if len(fr.calls) < 2 {
		t.Fatalf("expected docker calls, got %d", len(fr.calls))
	}
	if fr.calls[0].args[0] != "docker" || fr.calls[1].args[0] != "docker" {
		t.Fatalf("expected docker commands, got %+v", fr.calls)
	}
}

func TestEnsureUsesExistingContainer(t *testing.T) {
	fr := &fakeRunner{
		outputs: map[string]string{
			"docker ps -a --filter name=krellin-repo1 --format {{.ID}}": "abc123",
		},
	}
	c := New(fr)
	_, err := c.Ensure(context.Background(), capsule.Config{RepoID: "repo1", RepoRoot: "/repo", ImageDigest: "img@sha256:abc"})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	if len(fr.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fr.calls))
	}
}

func TestResetStopsRemovesCreates(t *testing.T) {
	fr := &fakeRunner{}
	c := New(fr)
	h := capsule.Handle{ID: "krellin-repo1", RepoID: "repo1", RepoRoot: "/repo"}
	err := c.Reset(context.Background(), h, "img@sha256:abc", true)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	if len(fr.calls) < 3 {
		t.Fatalf("expected at least 3 docker calls, got %d", len(fr.calls))
	}
}

func TestBuildCreateArgsSecurityFlags(t *testing.T) {
	args := buildCreateArgs(capsule.Config{RepoID: "repo1", RepoRoot: "/repo", ImageDigest: "img@sha256:abc", NetworkOn: true, CreatedAt: "2026-02-27T12:00:00Z"}, "krellin-repo1")
	joined := joinArgs(args)
	if !strings.Contains(joined, "--cap-drop=ALL") || !strings.Contains(joined, "--security-opt no-new-privileges") {
		t.Fatalf("expected security flags, got %s", joined)
	}
	if !strings.Contains(joined, "krellin.repo_id=repo1") || !strings.Contains(joined, "krellin.repo_root=/repo") || !strings.Contains(joined, "krellin.kind=capsule") {
		t.Fatalf("expected labels, got %s", joined)
	}
	if !strings.Contains(joined, "krellin.created_at=2026-02-27T12:00:00Z") {
		t.Fatalf("expected created_at label, got %s", joined)
	}
}

func TestSetNetwork(t *testing.T) {
	fr := &fakeRunner{}
	c := New(fr)
	h := capsule.Handle{ID: "krellin-repo1"}

	if err := c.SetNetwork(context.Background(), h, false); err != nil {
		t.Fatalf("disable network: %v", err)
	}
	if err := c.SetNetwork(context.Background(), h, true); err != nil {
		t.Fatalf("enable network: %v", err)
	}

	if len(fr.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(fr.calls))
	}
	if fr.calls[0].args[1] != "network" || fr.calls[1].args[1] != "network" {
		t.Fatalf("expected network commands, got %+v", fr.calls)
	}
}

func TestStatus(t *testing.T) {
	fr := &fakeRunner{outputs: map[string]string{
		"docker inspect -f {{.State.Running}}|{{.Config.Image}}|{{json .Config.Labels}} krellin-repo1": "true|img@sha256:abc|{\"krellin.repo_id\":\"repo1\"}",
	}}
	c := New(fr)
	status, err := c.Status(context.Background(), capsule.Handle{ID: "krellin-repo1"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.Running || status.Image != "img@sha256:abc" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if status.Labels["krellin.repo_id"] != "repo1" {
		t.Fatalf("unexpected labels: %+v", status.Labels)
	}
}

func TestCommitReturnsImageID(t *testing.T) {
	fr := &fakeRunner{outputs: map[string]string{
		"docker commit krellin-repo1": "sha256:abc",
	}}
	c := New(fr)
	id, err := c.Commit(context.Background(), capsule.Handle{ID: "krellin-repo1"}, capsule.CommitOptions{})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if id != "sha256:abc" {
		t.Fatalf("unexpected id: %q", id)
	}
}

type fakePTYFactory struct {
	container string
	conn      capsule.PTYConn
}

func (f *fakePTYFactory) Exec(ctx context.Context, containerID string) (capsule.PTYConn, error) {
	f.container = containerID
	return f.conn, nil
}

type fakePTYConn struct{}

func (f *fakePTYConn) Read(p []byte) (int, error)  { return 0, nil }
func (f *fakePTYConn) Write(p []byte) (int, error) { return len(p), nil }
func (f *fakePTYConn) Close() error                { return nil }

func TestAttachPTYUsesFactory(t *testing.T) {
	fr := &fakeRunner{}
	ptyFactory := &fakePTYFactory{conn: &fakePTYConn{}}
	c := NewWithPTY(fr, ptyFactory)

	conn, err := c.AttachPTY(context.Background(), capsule.Handle{ID: "krellin-repo1"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if conn == nil {
		t.Fatalf("expected conn")
	}
	if ptyFactory.container != "krellin-repo1" {
		t.Fatalf("expected container id passed to factory")
	}
}
