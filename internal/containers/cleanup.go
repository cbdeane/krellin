package containers

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type CleanupPolicy struct {
	KeepLastN       int
	DeleteOlderThan string
	DeleteUnpinned  bool
}

type clockFunc func() time.Time

func (i *Inventory) Cleanup(ctx context.Context, policy CleanupPolicy) error {
	if policy.KeepLastN < 0 {
		return fmt.Errorf("keep last n must be non-negative")
	}
	images, err := i.ListImages(ctx)
	if err != nil {
		return err
	}
	candidates := filterUnpinned(images, policy.DeleteUnpinned)
	if policy.DeleteOlderThan != "" {
		dur, err := time.ParseDuration(policy.DeleteOlderThan)
		if err != nil {
			return err
		}
		candidates = filterOlderThan(candidates, i.now().Add(-dur))
	}
	if policy.KeepLastN > 0 {
		candidates = dropNewest(candidates, policy.KeepLastN)
	}
	for _, img := range candidates {
		if _, err := i.runner.Run(ctx, "docker", "rmi", img.ID); err != nil {
			return err
		}
	}
	return nil
}

func (i *Inventory) now() time.Time {
	if i.clock != nil {
		return i.clock()
	}
	return time.Now().UTC()
}

func filterUnpinned(images []ImageInfo, onlyUnpinned bool) []ImageInfo {
	if !onlyUnpinned {
		return images
	}
	out := []ImageInfo{}
	for _, img := range images {
		if img.Labels["krellin.pinned"] != "true" {
			out = append(out, img)
		}
	}
	return out
}

func filterOlderThan(images []ImageInfo, cutoff time.Time) []ImageInfo {
	out := []ImageInfo{}
	for _, img := range images {
		ts, err := time.Parse(time.RFC3339, img.Labels["krellin.created_at"])
		if err != nil {
			continue
		}
		if ts.Before(cutoff) {
			out = append(out, img)
		}
	}
	return out
}

func dropNewest(images []ImageInfo, keep int) []ImageInfo {
	sort.Slice(images, func(i, j int) bool {
		return images[i].Labels["krellin.created_at"] > images[j].Labels["krellin.created_at"]
	})
	if len(images) <= keep {
		return nil
	}
	return images[keep:]
}
