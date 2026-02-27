package docker

import (
	"bytes"
	"context"
	"os/exec"
)

// ExecRunner runs docker CLI commands.
type ExecRunner struct{}

func (r ExecRunner) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
