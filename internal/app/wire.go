package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"krellin/internal/agents"
	dockercapsule "krellin/internal/capsule/docker"
	"krellin/internal/config"
	"krellin/internal/containers"
	"krellin/internal/images"
	"krellin/internal/patch"
	"krellin/internal/policy"
	"krellin/internal/pty/docker"
	"krellin/internal/repo"
	"krellin/internal/session"
)

const defaultImageTag = "ubuntu:latest"

// BuildSession wires a session for a repo root.
func BuildSession(ctx context.Context, repoRoot string) (*session.Session, error) {
	root, err := repo.ResolveRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(root, ".krellinrc")

	runner := dockercapsule.ExecRunner{}
	ptyFactory := docker.NewFactory()
	caps := dockercapsule.NewWithPTY(runner, ptyFactory)
	inv := containers.New(runner)
	resolver := images.NewResolver(runner)
	publisher := images.NewPublisher(runner)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Initialize default config pinned to digest.
			digest, err := resolver.ResolveDigest(ctx, defaultImageTag)
			if err != nil {
				return nil, err
			}
			cfg = config.DefaultConfig(digest)
			if err := config.Write(cfgPath, cfg); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	pol := policy.DefaultPolicy(root, os.Getenv("HOME"))
	pol.Network = policy.NetworkMode(cfg.Policy.Network)
	pol.Resources.CPUs = cfg.Resources.CPUs
	pol.Resources.MemoryMB = cfg.Resources.MemoryMB

	sess := session.New(session.Options{
		SessionID:       "", // assigned by daemon
		RepoRoot:        root,
		CapsuleName:     "krellin-" + repoID(root),
		Handler:         nil,
		Capsule:         caps,
		Policy:          pol,
		ImageDigest:     cfg.Capsule.Image,
		NetworkOn:       cfg.Policy.Network == config.NetworkOn,
		CPUs:            cfg.Resources.CPUs,
		MemoryMB:        cfg.Resources.MemoryMB,
		Inventory:       inv,
		Patches:         patch.NewBookkeeper(root),
		ConfigPath:      cfgPath,
		Resolver:        resolver,
		Updater:         session.DefaultConfigUpdater(),
		Publisher:       publisher,
		PublishTo:       cfg.Freeze.Publish,
		Platforms:       cfg.Freeze.Platforms,
		AgentsStore:     agents.NewStore(agents.DefaultPath()),
		AgentsSelection: agents.NewSelectionStore(agents.DefaultSelectionPath()),
		AgentsChecker:   agents.DefaultChecker{},
		AgentsRunner:    agents.HTTPRunner{},
		AgentsSecrets:   agents.NewKeyringStore(),
	})
	return sess, nil
}

func repoID(root string) string {
	id, err := repo.RepoID(root)
	if err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
