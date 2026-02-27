package session

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"krellin/internal/agents"
	"krellin/internal/protocol"
)

func TestSessionHandlerAgentsList(t *testing.T) {
	dir := t.TempDir()
	store := agents.NewStore(dir + "/providers.json")
	selection := agents.NewSelectionStore(dir + "/agents.json")
	t.Setenv("OPENAI_API_KEY", "ok")
	if err := store.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY", Enabled: true},
		{Name: "p2", Type: agents.ProviderGemini, Model: "gemini-2.0-flash", APIKeyEnv: "GEMINI_API_KEY", Enabled: true},
	}); err != nil {
		t.Fatalf("save providers: %v", err)
	}
	if err := selection.Save(agents.Selection{Active: "p2"}); err != nil {
		t.Fatalf("save selection: %v", err)
	}

	s := newBareSession(t, nil, nil)
	s.agentsStore = store
	s.agentsSelection = selection
	s.agentsChecker = fakeChecker{statusByName: map[string]string{"p1": "ready", "p2": "unreachable"}}
	s.agentsSecrets = &fakeSecrets{keys: map[string]string{"p1": "k"}}
	h := SessionHandler{Session: s}

	ch := s.Subscribe(1)
	defer s.Unsubscribe(ch)

	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentsList,
		Timestamp: time.Now(),
		Payload:   []byte(`{}`),
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	ev := <-ch
	if ev.Type != protocol.EventAgentsList {
		t.Fatalf("expected agents list event, got %s", ev.Type)
	}
	var payload protocol.AgentsListPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Active != "p2" {
		t.Fatalf("expected active p2, got %q", payload.Active)
	}
	if len(payload.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(payload.Providers))
	}
	if payload.Providers[0].Status != "ready" {
		t.Fatalf("expected p1 ready, got %q", payload.Providers[0].Status)
	}
	if !payload.Providers[0].HasAPIKey {
		t.Fatalf("expected p1 has_api_key")
	}
	if payload.Providers[1].Status != "missing_key" {
		t.Fatalf("expected p2 missing_key, got %q", payload.Providers[1].Status)
	}
}

func TestSessionHandlerAgentsSetActiveEnables(t *testing.T) {
	dir := t.TempDir()
	store := agents.NewStore(dir + "/providers.json")
	selection := agents.NewSelectionStore(dir + "/agents.json")
	if err := store.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY", Enabled: false},
	}); err != nil {
		t.Fatalf("save providers: %v", err)
	}

	s := newBareSession(t, nil, nil)
	s.agentsStore = store
	s.agentsSelection = selection
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.AgentsSetActivePayload{Name: "p1"})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentsSetActive,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	providers, err := store.Load()
	if err != nil {
		t.Fatalf("load providers: %v", err)
	}
	if !providers[0].Enabled {
		t.Fatalf("expected provider enabled")
	}
	sel, err := selection.Load()
	if err != nil {
		t.Fatalf("load selection: %v", err)
	}
	if sel.Active != "p1" {
		t.Fatalf("expected active p1, got %q", sel.Active)
	}
}

func TestSessionHandlerAgentsToggleDisablesActive(t *testing.T) {
	dir := t.TempDir()
	store := agents.NewStore(dir + "/providers.json")
	selection := agents.NewSelectionStore(dir + "/agents.json")
	if err := store.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY", Enabled: true},
	}); err != nil {
		t.Fatalf("save providers: %v", err)
	}
	if err := selection.Save(agents.Selection{Active: "p1"}); err != nil {
		t.Fatalf("save selection: %v", err)
	}

	s := newBareSession(t, nil, nil)
	s.agentsStore = store
	s.agentsSelection = selection
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.AgentsTogglePayload{Name: "p1", Enabled: false})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentsToggle,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	sel, err := selection.Load()
	if err != nil {
		t.Fatalf("load selection: %v", err)
	}
	if sel.Active != "" {
		t.Fatalf("expected active cleared, got %q", sel.Active)
	}
}

func TestSessionHandlerAgentsAdd(t *testing.T) {
	dir := t.TempDir()
	store := agents.NewStore(dir + "/providers.json")
	selection := agents.NewSelectionStore(dir + "/agents.json")
	secrets := &fakeSecrets{keys: map[string]string{}}

	s := newBareSession(t, nil, nil)
	s.agentsStore = store
	s.agentsSelection = selection
	s.agentsSecrets = secrets
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.AgentsAddPayload{
		Name:    "p1",
		Type:    "openai",
		Model:   "gpt-4o-mini",
		APIKey:  "sk-test",
		Enabled: true,
	})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentsAdd,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	providers, err := store.Load()
	if err != nil {
		t.Fatalf("load providers: %v", err)
	}
	if len(providers) != 1 || providers[0].Name != "p1" {
		t.Fatalf("unexpected providers: %+v", providers)
	}
	if secrets.keys["p1"] != "sk-test" {
		t.Fatalf("expected secret stored")
	}
}

func TestSessionHandlerPromptEmitsAgentMessage(t *testing.T) {
	s := newBareSession(t, nil, nil)
	s.agentsStore = agents.NewStore(t.TempDir() + "/providers.json")
	s.agentsSelection = agents.NewSelectionStore(t.TempDir() + "/agents.json")
	runner := &fakeRunner{response: "hi"}
	s.agentsRunner = runner
	s.agentsSecrets = &fakeSecrets{keys: map[string]string{"p1": "k"}}
	_ = s.agentsStore.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt", APIKeyEnv: "OPENAI_API_KEY", Enabled: true},
	})
	h := SessionHandler{Session: s}

	ch := s.Subscribe(1)
	defer s.Unsubscribe(ch)

	payload, _ := json.Marshal(protocol.AgentPromptPayload{Content: "hello"})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentPrompt,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	ev := <-ch
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent message, got %s", ev.Type)
	}
}

func TestSessionHandlerPromptUsesActiveProvider(t *testing.T) {
	s := newBareSession(t, nil, nil)
	s.agentsStore = agents.NewStore(t.TempDir() + "/providers.json")
	s.agentsSelection = agents.NewSelectionStore(t.TempDir() + "/agents.json")
	runner := &fakeRunner{response: "ok"}
	s.agentsRunner = runner
	s.agentsSecrets = &fakeSecrets{keys: map[string]string{"p2": "k"}}
	_ = s.agentsStore.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt", APIKeyEnv: "OPENAI_API_KEY", Enabled: true},
		{Name: "p2", Type: agents.ProviderAnthropic, Model: "claude", APIKeyEnv: "ANTHROPIC_API_KEY", Enabled: true},
	})
	_ = s.agentsSelection.Save(agents.Selection{Active: "p2"})
	h := SessionHandler{Session: s}

	payload, _ := json.Marshal(protocol.AgentPromptPayload{Content: "hello"})
	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentPrompt,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if runner.lastProvider != "p2" {
		t.Fatalf("expected p2 runner, got %q", runner.lastProvider)
	}
}

func TestSessionHandlerAgentsListUsesChecker(t *testing.T) {
	dir := t.TempDir()
	store := agents.NewStore(dir + "/providers.json")
	selection := agents.NewSelectionStore(dir + "/agents.json")
	t.Setenv("OPENAI_API_KEY", "ok")
	if err := store.Save([]agents.Provider{
		{Name: "p1", Type: agents.ProviderOpenAI, Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY", Enabled: true},
	}); err != nil {
		t.Fatalf("save providers: %v", err)
	}

	s := newBareSession(t, nil, nil)
	s.agentsStore = store
	s.agentsSelection = selection
	s.agentsChecker = fakeChecker{statusByName: map[string]string{"p1": "unreachable"}}
	h := SessionHandler{Session: s}

	ch := s.Subscribe(1)
	defer s.Unsubscribe(ch)

	action := protocol.Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      protocol.ActionAgentsList,
		Timestamp: time.Now(),
		Payload:   []byte(`{}`),
	}
	if err := h.Handle(context.Background(), action); err != nil {
		t.Fatalf("handle: %v", err)
	}

	ev := <-ch
	var payload protocol.AgentsListPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Providers[0].Status != "unreachable" {
		t.Fatalf("expected unreachable, got %q", payload.Providers[0].Status)
	}
}

type fakeChecker struct {
	statusByName map[string]string
}

func (f fakeChecker) Check(ctx context.Context, provider agents.Provider) string {
	if status, ok := f.statusByName[provider.Name]; ok {
		return status
	}
	return ""
}

type fakeRunner struct {
	response     string
	lastProvider string
}

func (f *fakeRunner) Prompt(ctx context.Context, provider agents.Provider, prompt string) (string, error) {
	f.lastProvider = provider.Name
	return f.response, nil
}

type fakeSecrets struct {
	keys map[string]string
}

func (f *fakeSecrets) Get(providerName string) (string, error) {
	val, ok := f.keys[providerName]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return val, nil
}

func (f *fakeSecrets) Set(providerName string, secret string) error {
	f.keys[providerName] = secret
	return nil
}

func (f *fakeSecrets) Delete(providerName string) error {
	delete(f.keys, providerName)
	return nil
}
