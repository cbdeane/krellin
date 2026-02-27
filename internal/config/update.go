package config

import "fmt"

// UpdateImageDigest loads the config at path, updates capsule.image, and writes it back.
func UpdateImageDigest(path string, digest string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	if !HasDigest(digest) {
		return fmt.Errorf("image must be pinned to a digest")
	}
	cfg.Capsule.Image = digest
	return Write(path, cfg)
}
