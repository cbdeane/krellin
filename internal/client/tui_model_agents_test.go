package client

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	tea "charm.land/bubbletea/v2"
	"krellin/internal/protocol"
)

type recordingClient struct {
	last []byte
}

func (r *recordingClient) SendAction(ctx context.Context, action []byte) error {
	r.last = action
	return nil
}
func (r *recordingClient) Subscribe(ctx context.Context) (<-chan []byte, error) {
	return make(chan []byte), nil
}

func TestAgentsCommandOpensModalAndRequestsList(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.input.SetValue("/agents")

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(*tuiModel)
	if !m.agentsOpen {
		t.Fatalf("expected agents modal open")
	}
	if cmd == nil {
		t.Fatalf("expected command")
	}
	_ = cmd()

	var action protocol.Action
	if err := json.Unmarshal(client.last, &action); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	if action.Type != protocol.ActionAgentsList {
		t.Fatalf("expected agents_list action, got %s", action.Type)
	}
}

func TestNaturalLanguageSendsPrompt(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.input.SetValue("hello there")
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(*tuiModel)
	if cmd == nil {
		t.Fatalf("expected command")
	}
	_ = cmd()
	var action protocol.Action
	if err := json.Unmarshal(client.last, &action); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	if action.Type != protocol.ActionAgentPrompt {
		t.Fatalf("expected agent_prompt action, got %s", action.Type)
	}
}

func TestAgentsModalRendersProviders(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.agentsOpen = true
	m.agentsProviders = []protocol.AgentProviderInfo{
		{Name: "p1", Type: "openai", Model: "gpt-4o-mini", Enabled: true},
	}
	view := m.View().Content
	if !bytes.Contains([]byte(view), []byte("Agents")) {
		t.Fatalf("expected Agents title in view")
	}
	if !bytes.Contains([]byte(view), []byte("p1")) {
		t.Fatalf("expected provider in view")
	}
}

func TestAgentsAddFlowSendsAction(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.agentsOpen = true
	m.agentsMode = "list"

	next, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = next.(*tuiModel)
	if m.agentsMode != "add" {
		t.Fatalf("expected add mode")
	}

	m.agentsAddFields[0].SetValue("p1")
	m.agentsAddFields[1].SetValue("openai")
	m.agentsAddFields[2].SetValue("gpt-4o-mini")
	m.agentsAddFields[3].SetValue("sk-test")
	m.agentsAddEnabled = true
	m.setAgentsAddField(len(m.agentsAddFields))

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(*tuiModel)
	if cmd == nil {
		t.Fatalf("expected submit command")
	}
	_ = cmd()

	var action protocol.Action
	if err := json.Unmarshal(client.last, &action); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	if action.Type != protocol.ActionAgentsAdd {
		t.Fatalf("expected agents_add action, got %s", action.Type)
	}
}

func TestAgentsAddValidationErrors(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.agentsOpen = true
	m.openAgentsAdd()
	m.setAgentsAddField(len(m.agentsAddFields))

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(*tuiModel)
	if cmd != nil {
		t.Fatalf("expected no command on validation error")
	}
	if m.agentsAddErr == "" {
		t.Fatalf("expected validation error")
	}
}

func TestAgentsEditPrefillsFields(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.agentsOpen = true
	m.agentsMode = "list"
	m.agentsProviders = []protocol.AgentProviderInfo{
		{Name: "p1", Type: "openai", Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY", BaseURL: "https://api.openai.com", Enabled: true},
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = next.(*tuiModel)
	if m.agentsMode != "add" || m.agentsAddMode != "edit" {
		t.Fatalf("expected edit mode")
	}
	if m.agentsAddFields[0].Value() != "p1" {
		t.Fatalf("expected name prefilled")
	}
	if m.agentsAddFields[2].Value() != "gpt-4o-mini" {
		t.Fatalf("expected model prefilled")
	}
}

func TestAgentsDeleteConfirmation(t *testing.T) {
	client := &recordingClient{}
	m := newTUIModel(client, "s1", "agent")
	m.agentsOpen = true
	m.agentsMode = "list"
	m.agentsProviders = []protocol.AgentProviderInfo{
		{Name: "p1", Type: "openai", Model: "gpt-4o-mini", Enabled: true},
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	m = next.(*tuiModel)
	if !m.agentsDeletePending {
		t.Fatalf("expected delete pending")
	}

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	m = next.(*tuiModel)
	if cmd == nil {
		t.Fatalf("expected delete command")
	}
	_ = cmd()

	var action protocol.Action
	if err := json.Unmarshal(client.last, &action); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	if action.Type != protocol.ActionAgentsDelete {
		t.Fatalf("expected agents_delete action, got %s", action.Type)
	}
}
