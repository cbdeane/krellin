package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleTOML = `version = 1

[capsule]
image = "ghcr.io/krellin/capsules/debian@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

[policy]
network = "on"

[resources]
cpus = 2
memory_mb = 4096

[freeze]
publish = ""
mode = "clean"
`

func TestParseConfigDefaults(t *testing.T) {
	data := []byte(`version = 1

[capsule]
image = "ghcr.io/krellin/capsules/debian@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if cfg.Policy.Network != NetworkOn {
		t.Fatalf("expected default network on, got %q", cfg.Policy.Network)
	}
	if cfg.Resources.CPUs != DefaultCPUs || cfg.Resources.MemoryMB != DefaultMemoryMB {
		t.Fatalf("unexpected resource defaults: %+v", cfg.Resources)
	}
	if cfg.Freeze.Mode != FreezeModeClean {
		t.Fatalf("expected default freeze mode clean, got %q", cfg.Freeze.Mode)
	}
}

func TestValidateDigestPinned(t *testing.T) {
	cfg := Config{
		Version: 1,
		Capsule: CapsuleConfig{Image: "ghcr.io/krellin/capsules/debian:latest"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error for non-digest image")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".krellinrc")

	cfg, err := Parse([]byte(sampleTOML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if err := Write(path, cfg); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	loaded, err := Parse(data)
	if err != nil {
		t.Fatalf("parse written: %v", err)
	}

	if loaded.Capsule.Image != cfg.Capsule.Image || loaded.Version != cfg.Version {
		t.Fatalf("round trip mismatch: %+v vs %+v", loaded, cfg)
	}
}
