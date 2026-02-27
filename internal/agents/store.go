package agents

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGrok      ProviderType = "grok"
	ProviderGemini    ProviderType = "gemini"
	ProviderLLaMA     ProviderType = "llama"
)

type Provider struct {
	Name      string       `json:"name"`
	Type      ProviderType `json:"type"`
	BaseURL   string       `json:"base_url,omitempty"`
	Model     string       `json:"model"`
	APIKeyEnv string       `json:"api_key_env"`
	APIKey    string       `json:"-"`
	Enabled   bool         `json:"enabled"`
	Notes     string       `json:"notes,omitempty"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func DefaultPath() string {
	root := os.Getenv("KRELLIN_HOME")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".krellin")
	}
	return filepath.Join(root, "providers.json")
}

func (s *Store) Load() ([]Provider, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Provider{}, nil
		}
		return nil, err
	}
	var providers []Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *Store) Save(providers []Provider) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Upsert(p Provider) error {
	providers, err := s.Load()
	if err != nil {
		return err
	}
	updated := false
	for i := range providers {
		if providers[i].Name == p.Name {
			providers[i] = p
			updated = true
			break
		}
	}
	if !updated {
		providers = append(providers, p)
	}
	return s.Save(providers)
}
