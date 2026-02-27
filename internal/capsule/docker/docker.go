package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"krellin/internal/capsule"
)

type Runner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

type PTYFactory interface {
	Exec(ctx context.Context, containerID string) (capsule.PTYConn, error)
}

type Capsule struct {
	runner     Runner
	ptyFactory PTYFactory
}

func New(runner Runner) *Capsule {
	return &Capsule{runner: runner}
}

func NewWithPTY(runner Runner, ptyFactory PTYFactory) *Capsule {
	return &Capsule{runner: runner, ptyFactory: ptyFactory}
}

func (c *Capsule) Ensure(ctx context.Context, cfg capsule.Config) (capsule.Handle, error) {
	name := containerName(cfg.RepoID)
	existing, err := c.runner.Run(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.ID}}")
	if err != nil {
		return capsule.Handle{}, err
	}
	if strings.TrimSpace(existing) != "" {
		return capsule.Handle{ID: name, RepoID: cfg.RepoID, RepoRoot: cfg.RepoRoot}, nil
	}

	if _, err := c.runner.Run(ctx, buildCreateArgs(cfg, name)...); err != nil {
		return capsule.Handle{}, err
	}
	if _, err := c.runner.Run(ctx, "docker", "start", name); err != nil {
		return capsule.Handle{}, err
	}
	return capsule.Handle{ID: name, RepoID: cfg.RepoID, RepoRoot: cfg.RepoRoot}, nil
}

func (c *Capsule) Start(ctx context.Context, handle capsule.Handle) error {
	_, err := c.runner.Run(ctx, "docker", "start", handle.ID)
	return err
}

func (c *Capsule) Stop(ctx context.Context, handle capsule.Handle) error {
	_, err := c.runner.Run(ctx, "docker", "stop", handle.ID)
	return err
}

func (c *Capsule) Reset(ctx context.Context, handle capsule.Handle, imageDigest string, preserveVolumes bool) error {
	_, _ = c.runner.Run(ctx, "docker", "stop", handle.ID)
	_, _ = c.runner.Run(ctx, "docker", "rm", "-f", handle.ID)
	if handle.RepoRoot == "" {
		return fmt.Errorf("repo root required for reset")
	}
	repoID := handle.RepoID
	if repoID == "" {
		repoID = strings.TrimPrefix(handle.ID, "krellin-")
	}
	cfg := capsule.Config{RepoID: repoID, RepoRoot: handle.RepoRoot, ImageDigest: imageDigest}
	_, err := c.runner.Run(ctx, buildCreateArgs(cfg, handle.ID)...)
	return err
}

func (c *Capsule) AttachPTY(ctx context.Context, handle capsule.Handle) (capsule.PTYConn, error) {
	if c.ptyFactory == nil {
		return nil, fmt.Errorf("pty factory not configured")
	}
	return c.ptyFactory.Exec(ctx, handle.ID)
}

func (c *Capsule) Commit(ctx context.Context, handle capsule.Handle, opts capsule.CommitOptions) (string, error) {
	out, err := c.runner.Run(ctx, "docker", "commit", handle.ID)
	if err != nil {
		return "", err
	}
	imageID := strings.TrimSpace(out)
	if imageID == "" {
		return "", fmt.Errorf("commit returned empty image id")
	}
	return imageID, nil
}

func (c *Capsule) SetNetwork(ctx context.Context, handle capsule.Handle, enabled bool) error {
	if enabled {
		_, err := c.runner.Run(ctx, "docker", "network", "connect", "bridge", handle.ID)
		return err
	}
	_, err := c.runner.Run(ctx, "docker", "network", "disconnect", "bridge", handle.ID)
	return err
}

func (c *Capsule) Status(ctx context.Context, handle capsule.Handle) (capsule.Status, error) {
	out, err := c.runner.Run(ctx, "docker", "inspect", "-f", "{{.State.Running}}|{{.Config.Image}}|{{json .Config.Labels}}", handle.ID)
	if err != nil {
		return capsule.Status{}, err
	}
	parts := strings.SplitN(strings.TrimSpace(out), "|", 3)
	if len(parts) < 2 {
		return capsule.Status{}, fmt.Errorf("unexpected inspect output")
	}
	labels := map[string]string{}
	if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
		if err := json.Unmarshal([]byte(parts[2]), &labels); err != nil {
			return capsule.Status{}, fmt.Errorf("invalid labels output")
		}
	}
	return capsule.Status{
		Running: parts[0] == "true",
		Image:   parts[1],
		Labels:  labels,
	}, nil
}

func containerName(repoID string) string {
	return fmt.Sprintf("krellin-%s", repoID)
}

func buildCreateArgs(cfg capsule.Config, name string) []string {
	args := []string{"docker", "create", "--name", name}
	args = append(args, "--cap-drop=ALL")
	args = append(args, "--security-opt", "no-new-privileges")
	args = append(args, "--label", fmt.Sprintf("krellin.repo_id=%s", cfg.RepoID))
	args = append(args, "--label", fmt.Sprintf("krellin.repo_root=%s", cfg.RepoRoot))
	args = append(args, "--label", "krellin.kind=capsule")
	if cfg.CreatedAt != "" {
		args = append(args, "--label", fmt.Sprintf("krellin.created_at=%s", cfg.CreatedAt))
	}
	args = append(args, "-v", fmt.Sprintf("krellin-%s-home:/home/dev", cfg.RepoID))
	args = append(args, "-v", fmt.Sprintf("krellin-%s-env:/env", cfg.RepoID))
	args = append(args, "-v", fmt.Sprintf("%s:/workspace", cfg.RepoRoot))
	if !cfg.NetworkOn {
		args = append(args, "--network", "none")
	}
	if cfg.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%d", cfg.CPUs))
	}
	if cfg.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", cfg.MemoryMB))
	}
	args = append(args, cfg.ImageDigest, "sleep", "infinity")
	return args
}
