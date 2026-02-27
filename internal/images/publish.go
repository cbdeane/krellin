package images

import (
	"context"
	"fmt"
)

type Publisher struct {
	runner Runner
}

func NewPublisher(runner Runner) *Publisher {
	return &Publisher{runner: runner}
}

// Push tags the image to the publish target and pushes it.
func (p *Publisher) Push(ctx context.Context, imageRef string, target string, platforms []string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("publish target required")
	}
	if _, err := p.runner.Run(ctx, "docker", "tag", imageRef, target); err != nil {
		return "", err
	}
	args := []string{"docker", "push"}
	if len(platforms) > 0 {
		args = append(args, "--platform", joinPlatforms(platforms))
	}
	args = append(args, target)
	if _, err := p.runner.Run(ctx, args...); err != nil {
		return "", err
	}
	return target, nil
}

func joinPlatforms(platforms []string) string {
	if len(platforms) == 0 {
		return ""
	}
	out := platforms[0]
	for i := 1; i < len(platforms); i++ {
		out += "," + platforms[i]
	}
	return out
}
