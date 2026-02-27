package config

// DefaultConfig returns a baseline config for a repo.
func DefaultConfig(imageDigest string) Config {
	cfg := Config{
		Version: 1,
		Capsule: CapsuleConfig{Image: imageDigest},
		Policy:  PolicyConfig{Network: NetworkOn},
		Resources: ResourcesConfig{
			CPUs:     DefaultCPUs,
			MemoryMB: DefaultMemoryMB,
		},
		Freeze: FreezeConfig{Mode: FreezeModeClean},
	}
	return cfg
}
