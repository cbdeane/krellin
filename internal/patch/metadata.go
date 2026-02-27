package patch

import (
	"path/filepath"
	"time"
)

type Metadata struct {
	Timestamp time.Time
	Files     []string
	Patch     string
}

func (b *Bookkeeper) LastMetadata() Metadata {
	return Metadata{Timestamp: time.Now().UTC(), Files: append([]string{}, b.lastFiles...), Patch: b.lastPatch}
}

func (b *Bookkeeper) CheckpointPath() string {
	return filepath.Join(b.root, ".krellin", "checkpoints")
}
