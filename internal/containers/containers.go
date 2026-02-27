package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

type CapsuleInfo struct {
	ID     string
	Name   string
	RepoID string
	Labels map[string]string
	SizeBytes int64
}

type Inventory struct {
	runner Runner
	clock  clockFunc
}

func New(runner Runner) *Inventory {
	return &Inventory{runner: runner}
}

func NewWithClock(runner Runner, clock clockFunc) *Inventory {
	return &Inventory{runner: runner, clock: clock}
}

func (i *Inventory) ListCapsules(ctx context.Context) ([]CapsuleInfo, error) {
	out, err := i.runner.Run(ctx, "docker", "ps", "-a", "--filter", "label=krellin.kind=capsule", "--format", "{{.ID}}|{{.Names}}")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	items := make([]CapsuleInfo, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid docker ps output")
		}
		labels, err := i.inspectLabels(ctx, parts[0])
		if err != nil {
			return nil, err
		}
		size, _ := i.inspectContainerSize(ctx, parts[0])
		items = append(items, CapsuleInfo{ID: parts[0], Name: parts[1], RepoID: labels["krellin.repo_id"], Labels: labels, SizeBytes: size})
	}
	return items, nil
}

func (i *Inventory) ListImages(ctx context.Context) ([]ImageInfo, error) {
	out, err := i.runner.Run(ctx, "docker", "images", "--filter", "label=krellin.kind", "--format", "{{.ID}}|{{.Repository}}|{{.Tag}}")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	items := make([]ImageInfo, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid docker images output")
		}
		labels, err := i.inspectImageLabels(ctx, parts[0])
		if err != nil {
			return nil, err
		}
		size, _ := i.inspectImageSize(ctx, parts[0])
		items = append(items, ImageInfo{ID: parts[0], Repository: parts[1], Tag: parts[2], Labels: labels, SizeBytes: size})
	}
	return items, nil
}

func (i *Inventory) inspectLabels(ctx context.Context, id string) (map[string]string, error) {
	out, err := i.runner.Run(ctx, "docker", "inspect", "-f", "{{json .Config.Labels}}", id)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &labels); err != nil {
		return nil, fmt.Errorf("invalid labels output")
	}
	return labels, nil
}

func (i *Inventory) inspectImageLabels(ctx context.Context, id string) (map[string]string, error) {
	out, err := i.runner.Run(ctx, "docker", "inspect", "-f", "{{json .Config.Labels}}", id)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &labels); err != nil {
		return nil, fmt.Errorf("invalid labels output")
	}
	return labels, nil
}

func (i *Inventory) inspectContainerSize(ctx context.Context, id string) (int64, error) {
	out, err := i.runner.Run(ctx, "docker", "inspect", "-f", "{{.SizeRw}}", id)
	if err != nil {
		return 0, err
	}
	return parseInt64(strings.TrimSpace(out))
}

func (i *Inventory) inspectImageSize(ctx context.Context, id string) (int64, error) {
	out, err := i.runner.Run(ctx, "docker", "inspect", "-f", "{{.Size}}", id)
	if err != nil {
		return 0, err
	}
	return parseInt64(strings.TrimSpace(out))
}

func parseInt64(val string) (int64, error) {
	if val == "" {
		return 0, fmt.Errorf("empty size")
	}
	var num int64
	_, err := fmt.Sscanf(val, "%d", &num)
	return num, err
}

type ImageInfo struct {
	ID         string
	Repository string
	Tag        string
	Labels     map[string]string
	SizeBytes  int64
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
