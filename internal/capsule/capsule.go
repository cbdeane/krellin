package capsule

import "context"

type Handle struct {
	ID       string
	RepoID   string
	RepoRoot string
}

type CommitOptions struct {
	Mode string
}

type Status struct {
	Running bool
	Image   string
	Labels  map[string]string
}

type Config struct {
	RepoID      string
	RepoRoot    string
	ImageDigest string
	NetworkOn   bool
	CPUs        int
	MemoryMB    int
	CreatedAt   string
}

type PTYConn interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
}

type Capsule interface {
	Ensure(ctx context.Context, cfg Config) (Handle, error)
	Start(ctx context.Context, handle Handle) error
	Stop(ctx context.Context, handle Handle) error
	Reset(ctx context.Context, handle Handle, imageDigest string, preserveVolumes bool) error
	AttachPTY(ctx context.Context, handle Handle) (PTYConn, error)
	Commit(ctx context.Context, handle Handle, opts CommitOptions) (string, error)
	SetNetwork(ctx context.Context, handle Handle, enabled bool) error
	Status(ctx context.Context, handle Handle) (Status, error)
}
