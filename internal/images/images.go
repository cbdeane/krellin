package images

import (
	"context"
	"fmt"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

type Resolver struct {
	runner Runner
}

func NewResolver(runner Runner) *Resolver {
	return &Resolver{runner: runner}
}

func (r *Resolver) ResolveDigest(ctx context.Context, imageRef string) (string, error) {
	out, err := r.runner.Run(ctx, "docker", "inspect", "-f", "{{index .RepoDigests 0}}", imageRef)
	if err != nil {
		// Try pulling the image if inspect fails (likely missing locally).
		if _, pullErr := r.runner.Run(ctx, "docker", "pull", imageRef); pullErr != nil {
			return "", err
		}
		out, err = r.runner.Run(ctx, "docker", "inspect", "-f", "{{index .RepoDigests 0}}", imageRef)
		if err != nil {
			return "", err
		}
	}
	res := strings.TrimSpace(out)
	if res == "" {
		return "", fmt.Errorf("empty digest")
	}
	return res, nil
}
