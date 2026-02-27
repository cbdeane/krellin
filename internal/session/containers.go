package session

import (
	"context"

	"krellin/internal/containers"
)

type ContainersInventory interface {
	ListCapsules(ctx context.Context) ([]containers.CapsuleInfo, error)
	ListImages(ctx context.Context) ([]containers.ImageInfo, error)
}
