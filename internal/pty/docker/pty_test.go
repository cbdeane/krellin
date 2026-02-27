package docker

import (
	"context"
	"os/exec"
	"testing"

	"krellin/internal/capsule"
)

type fakeExecFactory struct {
	name string
	args []string
}

func (f *fakeExecFactory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	f.name = name
	f.args = args
	return exec.CommandContext(ctx, "true")
}

type fakePTYConn struct{}

func (f *fakePTYConn) Read(p []byte) (int, error)  { return 0, nil }
func (f *fakePTYConn) Write(p []byte) (int, error) { return len(p), nil }
func (f *fakePTYConn) Close() error                { return nil }

func TestFactoryExecArgs(t *testing.T) {
	fx := &fakeExecFactory{}
	starter := func(cmd *exec.Cmd) (capsule.PTYConn, error) { return &fakePTYConn{}, nil }
	factory := NewFactoryWithStarter(fx, starter)

	conn, err := factory.Exec(context.Background(), "krellin-repo1")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if conn == nil {
		t.Fatalf("expected conn")
	}
	if fx.name != "docker" {
		t.Fatalf("expected docker command")
	}
	expected := []string{"exec", "-it", "krellin-repo1", "sh"}
	if len(fx.args) != len(expected) {
		t.Fatalf("unexpected args: %v", fx.args)
	}
	for i, arg := range expected {
		if fx.args[i] != arg {
			t.Fatalf("unexpected arg %d: %q", i, fx.args[i])
		}
	}
}
