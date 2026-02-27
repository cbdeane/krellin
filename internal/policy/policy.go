package policy

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	DefaultCPUs     = 2
	DefaultMemoryMB = 4096
)

type NetworkMode string

const (
	NetworkOn  NetworkMode = "on"
	NetworkOff NetworkMode = "off"
)

type Resources struct {
	CPUs     int
	MemoryMB int
}

type Policy struct {
	RepoRoot string
	HomeDir  string
	Network  NetworkMode
	Resources Resources
	Unsafe   bool
}

func DefaultPolicy(repoRoot, homeDir string) Policy {
	return Policy{
		RepoRoot: repoRoot,
		HomeDir:  homeDir,
		Network:  NetworkOn,
		Resources: Resources{
			CPUs:     DefaultCPUs,
			MemoryMB: DefaultMemoryMB,
		},
		Unsafe: false,
	}
}

func ValidateMount(policy Policy, hostPath string) error {
	if policy.Unsafe {
		return nil
	}
	clean := filepath.Clean(hostPath)
	if clean == "/" {
		return errors.New("mounting root is forbidden")
	}
	if strings.HasPrefix(clean, "/var/run/docker.sock") {
		return errors.New("mounting docker socket is forbidden")
	}
	if policy.HomeDir != "" && sameOrChild(clean, policy.HomeDir) {
		return errors.New("mounting home directory is forbidden")
	}
	if policy.RepoRoot == "" {
		return errors.New("repo root is required for mount validation")
	}
	if !sameOrChild(clean, policy.RepoRoot) {
		return errors.New("mount outside repo root is forbidden")
	}
	return nil
}

func sameOrChild(path, root string) bool {
	rootClean := filepath.Clean(root)
	pathClean := filepath.Clean(path)
	if pathClean == rootClean {
		return true
	}
	if !strings.HasSuffix(rootClean, string(filepath.Separator)) {
		rootClean += string(filepath.Separator)
	}
	return strings.HasPrefix(pathClean+string(filepath.Separator), rootClean)
}
