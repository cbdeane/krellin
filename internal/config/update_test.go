package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateImageDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".krellinrc")
	cfg := Config{Version: 1, Capsule: CapsuleConfig{Image: "repo@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", User: "root"}}
	if err := Write(path, cfg); err != nil {
		t.Fatalf("write: %v", err)
	}

	digest := "repo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := UpdateImageDigest(path, digest); err != nil {
		t.Fatalf("update: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	loaded, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if loaded.Capsule.Image != digest {
		t.Fatalf("unexpected digest: %q", loaded.Capsule.Image)
	}
}
