package agents

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Selection struct {
	Active string `json:"active"`
}

type SelectionStore struct {
	path string
}

func NewSelectionStore(path string) *SelectionStore {
	return &SelectionStore{path: path}
}

func DefaultSelectionPath() string {
	root := os.Getenv("KRELLIN_HOME")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".krellin")
	}
	return filepath.Join(root, "agents.json")
}

func (s *SelectionStore) Load() (Selection, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Selection{}, nil
		}
		return Selection{}, err
	}
	var sel Selection
	if err := json.Unmarshal(data, &sel); err != nil {
		return Selection{}, err
	}
	return sel, nil
}

func (s *SelectionStore) Save(sel Selection) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sel, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
