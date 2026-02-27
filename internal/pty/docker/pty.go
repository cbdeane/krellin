package docker

import (
	"context"
	"os/exec"

	"github.com/creack/pty"
	"krellin/internal/capsule"
)

// ExecFactory creates an exec.Cmd for docker exec.
type ExecFactory interface {
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd
}

type defaultExecFactory struct{}

func (d defaultExecFactory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// Factory attaches a PTY by running `docker exec -it`.
type Factory struct {
	execFactory ExecFactory
	ptyStarter  func(cmd *exec.Cmd) (capsule.PTYConn, error)
}

func NewFactory() *Factory {
	return &Factory{execFactory: defaultExecFactory{}, ptyStarter: startPTY}
}

func NewFactoryWithExec(execFactory ExecFactory) *Factory {
	return &Factory{execFactory: execFactory, ptyStarter: startPTY}
}

func NewFactoryWithStarter(execFactory ExecFactory, starter func(cmd *exec.Cmd) (capsule.PTYConn, error)) *Factory {
	return &Factory{execFactory: execFactory, ptyStarter: starter}
}

func (f *Factory) Exec(ctx context.Context, containerID string) (capsule.PTYConn, error) {
	cmd := f.execFactory.CommandContext(
		ctx,
		"docker",
		"exec",
		"-it",
		"-e", "PS1=",
		"-e", "PROMPT_COMMAND=",
		containerID,
		"sh",
	)
	return f.ptyStarter(cmd)
}

func startPTY(cmd *exec.Cmd) (capsule.PTYConn, error) {
	return pty.Start(cmd)
}
