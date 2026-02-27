package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreUpsertLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	store := NewStore(path)

	p := Provider{Name: "default", Type: ProviderOpenAI, Model: "gpt", APIKeyEnv: "OPENAI_API_KEY", Enabled: true}
	if err := store.Upsert(p); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	providers, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(providers) != 1 || providers[0].Name != "default" {
		t.Fatalf("unexpected providers: %+v", providers)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file: %v", err)
	}
}
