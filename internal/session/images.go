package session

import (
	"context"

	"krellin/internal/config"
)

type ImageResolver interface {
	ResolveDigest(ctx context.Context, imageRef string) (string, error)
}

type ImagePublisher interface {
	Push(ctx context.Context, imageRef string, target string, platforms []string) (string, error)
}

type ConfigUpdater interface {
	UpdateImage(path string, digest string) error
}

type ConfigUpdaterFunc func(path string, digest string) error

func (f ConfigUpdaterFunc) UpdateImage(path string, digest string) error {
	return f(path, digest)
}

func DefaultConfigUpdater() ConfigUpdater {
	return ConfigUpdaterFunc(config.UpdateImageDigest)
}
