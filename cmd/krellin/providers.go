package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"krellin/internal/agents"
)

type providersCmd struct{}

func (p providersCmd) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: krellin providers <list|add>")
	}
	store := agents.NewStore(agents.DefaultPath())
	switch args[0] {
	case "list":
		providers, err := store.Load()
		if err != nil {
			return err
		}
		if len(providers) == 0 {
			fmt.Println("No providers configured.")
			return nil
		}
		for _, prov := range providers {
			status := "disabled"
			if prov.Enabled {
				status = "enabled"
			}
			fmt.Printf("%s (%s) model=%s env=%s %s\n", prov.Name, prov.Type, prov.Model, prov.APIKeyEnv, status)
		}
		return nil
	case "add":
		fs := flag.NewFlagSet("providers add", flag.ContinueOnError)
		name := fs.String("name", "", "provider name")
		ptype := fs.String("type", "", "openai|anthropic|grok|gemini|llama")
		model := fs.String("model", "", "model name")
		baseURL := fs.String("base-url", "", "base url (optional)")
		keyEnv := fs.String("api-key-env", "", "env var for api key")
		enabled := fs.Bool("enabled", true, "enabled")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *name == "" || *ptype == "" || *model == "" || *keyEnv == "" {
			return errors.New("name, type, model, and api-key-env are required")
		}
		pt := agents.ProviderType(strings.ToLower(*ptype))
		if pt != agents.ProviderOpenAI && pt != agents.ProviderAnthropic && pt != agents.ProviderGrok && pt != agents.ProviderGemini && pt != agents.ProviderLLaMA {
			return errors.New("invalid provider type")
		}
		prov := agents.Provider{
			Name:      *name,
			Type:      pt,
			Model:     *model,
			BaseURL:   *baseURL,
			APIKeyEnv: *keyEnv,
			Enabled:   *enabled,
		}
		if err := store.Upsert(prov); err != nil {
			return err
		}
		fmt.Println("Provider saved.")
		return nil
	default:
		return errors.New("unknown providers command")
	}
}

func isProvidersCommand(args []string) bool {
	return len(args) > 0 && args[0] == "providers"
}

func runProviders(args []string) {
	cmd := providersCmd{}
	if err := cmd.Run(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
