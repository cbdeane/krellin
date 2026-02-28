package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
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

type FreezeMode string

const (
	FreezeModeClean FreezeMode = "clean"
	FreezeModeAsIs  FreezeMode = "as-is"
)

type Config struct {
	Version   int             `toml:"version"`
	Capsule   CapsuleConfig   `toml:"capsule"`
	Policy    PolicyConfig    `toml:"policy"`
	Resources ResourcesConfig `toml:"resources"`
	Freeze    FreezeConfig    `toml:"freeze"`
}

type CapsuleConfig struct {
	Image string `toml:"image"`
	Name  string `toml:"name,omitempty"`
	User  string `toml:"user,omitempty"`
}

type PolicyConfig struct {
	Network NetworkMode `toml:"network"`
}

type ResourcesConfig struct {
	CPUs     int `toml:"cpus"`
	MemoryMB int `toml:"memory_mb"`
}

type FreezeConfig struct {
	Publish   string     `toml:"publish"`
	Platforms []string   `toml:"platforms"`
	Mode      FreezeMode `toml:"mode"`
}

func Parse(data []byte) (Config, error) {
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	applyDefaults(&cfg)
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(data)
}

func Write(path string, cfg Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Validate(cfg Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", cfg.Version)
	}
	if cfg.Capsule.Image == "" {
		return errors.New("capsule.image is required")
	}
	if !HasDigest(cfg.Capsule.Image) {
		return errors.New("capsule.image must be pinned to a digest")
	}
	if cfg.Policy.Network != "" && cfg.Policy.Network != NetworkOn && cfg.Policy.Network != NetworkOff {
		return fmt.Errorf("invalid policy.network: %q", cfg.Policy.Network)
	}
	if cfg.Resources.CPUs < 0 || cfg.Resources.MemoryMB < 0 {
		return errors.New("resources values must be non-negative")
	}
	if cfg.Capsule.User != "" {
		if strings.TrimSpace(cfg.Capsule.User) == "" {
			return errors.New("capsule.user must be non-empty when set")
		}
	}
	if cfg.Freeze.Mode != "" && cfg.Freeze.Mode != FreezeModeClean && cfg.Freeze.Mode != FreezeModeAsIs {
		return fmt.Errorf("invalid freeze.mode: %q", cfg.Freeze.Mode)
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Policy.Network == "" {
		cfg.Policy.Network = NetworkOn
	}
	if cfg.Resources.CPUs == 0 {
		cfg.Resources.CPUs = DefaultCPUs
	}
	if cfg.Resources.MemoryMB == 0 {
		cfg.Resources.MemoryMB = DefaultMemoryMB
	}
	if cfg.Freeze.Mode == "" {
		cfg.Freeze.Mode = FreezeModeClean
	}
}

func HasDigest(image string) bool {
	idx := strings.Index(image, "@sha256:")
	if idx == -1 {
		return false
	}
	parts := strings.Split(image[idx+1:], ":")
	if len(parts) != 2 {
		return false
	}
	digest := parts[1]
	if len(digest) != 64 {
		return false
	}
	for _, c := range digest {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
