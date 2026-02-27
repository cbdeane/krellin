package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"krellin/internal/agents"
	"krellin/internal/capsule"
	"krellin/internal/containers"
	"krellin/internal/protocol"
)

// SessionHandler routes actions to the capsule backend.
type SessionHandler struct {
	Session *Session
}

func (h SessionHandler) Handle(ctx context.Context, action protocol.Action) error {
	s := h.Session
	if s == nil {
		return fmt.Errorf("capsule not configured")
	}
	switch action.Type {
	case protocol.ActionNetworkToggle:
		var payload protocol.NetworkTogglePayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid network_toggle payload")
		}
		return s.capsule.SetNetwork(ctx, s.handle, payload.Enabled)
	case protocol.ActionReset:
		var payload protocol.ResetPayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid reset payload")
		}
		if err := s.capsule.Reset(ctx, s.handle, s.imageDigest, payload.PreserveVolumes); err != nil {
			return err
		}
		s.resetPTY()
		_ = s.ensurePTY(ctx)
		banner, _ := protocol.MarshalPayload(protocol.TerminalOutputPayload{
			Stream: "stdout",
			Data:   "Krellin reset complete\n",
		})
		s.Emit(protocol.Event{
			EventID:   "terminal-banner",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventTerminalOutput,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   banner,
		})
		data, _ := protocol.MarshalPayload(struct{}{})
		s.Emit(protocol.Event{
			EventID:   "reset-completed",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventResetCompleted,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   data,
		})
		return nil
	case protocol.ActionFreeze:
		var payload protocol.FreezePayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid freeze payload")
		}
		imageRef, err := s.capsule.Commit(ctx, s.handle, capsule.CommitOptions{Mode: payload.Mode})
		if err != nil {
			return err
		}
		if s.publisher != nil && s.publishTo != "" {
			published, err := s.publisher.Push(ctx, imageRef, s.publishTo, s.platforms)
			if err != nil {
				return err
			}
			imageRef = published
		}
		digest := imageRef
		if s.resolver != nil {
			if resolved, err := s.resolver.ResolveDigest(ctx, imageRef); err == nil {
				digest = resolved
			} else {
				return err
			}
		}
		if s.updater != nil && s.configPath != "" {
			if err := s.updater.UpdateImage(s.configPath, digest); err != nil {
				return err
			}
		}
		payloadOut := protocol.FreezeCreatedPayload{Image: digest, SizeBytes: 0}
		data, _ := protocol.MarshalPayload(payloadOut)
		s.Emit(protocol.Event{
			EventID:   "freeze-created",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventFreezeCreated,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   data,
		})
		return nil
	case protocol.ActionRunCommand:
		var payload protocol.RunCommandPayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid run_command payload")
		}
		if s.pty == nil {
			if err := s.ensurePTY(ctx); err != nil {
				return err
			}
		}
		cmd := payload.Command
		if cmd == "" {
			return fmt.Errorf("command required")
		}
		data := []byte(cmd + "\n")
		for len(data) > 0 {
			n, err := s.pty.Write(data)
			if err != nil {
				return err
			}
			data = data[n:]
		}
		return nil
	case protocol.ActionApplyPatch:
		var payload protocol.ApplyPatchPayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid apply_patch payload")
		}
		if s.patches == nil {
			return fmt.Errorf("patches not configured")
		}
		_, err := s.patches.Apply(payload.Patch)
		if err != nil {
			return err
		}
		diffText, files, err := s.patches.Diff()
		if err != nil {
			return err
		}
		diffPayload := protocol.DiffReadyPayload{Patch: diffText, Files: files}
		data, _ := protocol.MarshalPayload(diffPayload)
		s.Emit(protocol.Event{
			EventID:   "diff-ready",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventDiffReady,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   data,
		})
		return nil
	case protocol.ActionRevert:
		if s.patches == nil {
			return fmt.Errorf("patches not configured")
		}
		return s.patches.Revert()
	case protocol.ActionContainersList:
		if s.inventory == nil {
			return fmt.Errorf("containers inventory not configured")
		}
		items, err := s.inventory.ListCapsules(ctx)
		if err != nil {
			return err
		}
		payload := protocol.ContainersListPayload{Capsules: toProtocolContainers(items)}
		data, _ := protocol.MarshalPayload(payload)
		s.Emit(protocol.Event{
			EventID:   "containers-list",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventContainersList,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   data,
		})
		return nil
	case protocol.ActionAgentsList:
		return h.emitAgentsList(ctx, action)
	case protocol.ActionAgentsSetActive:
		var payload protocol.AgentsSetActivePayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid agents_set_active payload")
		}
		return h.setActiveAgent(ctx, action, payload.Name)
	case protocol.ActionAgentsToggle:
		var payload protocol.AgentsTogglePayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid agents_toggle payload")
		}
		return h.toggleAgent(ctx, action, payload.Name, payload.Enabled)
	case protocol.ActionAgentsAdd:
		var payload protocol.AgentsAddPayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid agents_add payload")
		}
		return h.addAgent(ctx, action, payload)
	case protocol.ActionAgentsDelete:
		var payload protocol.AgentsDeletePayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid agents_delete payload")
		}
		return h.deleteAgent(ctx, action, payload.Name)
	case protocol.ActionAgentPrompt:
		var payload protocol.AgentPromptPayload
		if err := json.Unmarshal(action.Payload, &payload); err != nil {
			return fmt.Errorf("invalid agent_prompt payload")
		}
		return h.handleAgentPrompt(ctx, action, payload.Content)
	default:
		return fmt.Errorf("action not implemented")
	}
}

func (h SessionHandler) emitAgentsList(ctx context.Context, action protocol.Action) error {
	providers, active, err := h.loadAgents(ctx)
	if err != nil {
		return err
	}
	payload := protocol.AgentsListPayload{Providers: providers, Active: active}
	data, _ := protocol.MarshalPayload(payload)
	h.Session.Emit(protocol.Event{
		EventID:   "agents-list",
		SessionID: action.SessionID,
		Timestamp: time.Now().UTC(),
		Type:      protocol.EventAgentsList,
		Source:    protocol.SourceSystem,
		AgentID:   action.AgentID,
		Payload:   data,
	})
	return nil
}

func (h SessionHandler) setActiveAgent(ctx context.Context, action protocol.Action, name string) error {
	if name == "" {
		return fmt.Errorf("agent name required")
	}
	store, selection, err := h.agentStores()
	if err != nil {
		return err
	}
	providers, err := store.Load()
	if err != nil {
		return err
	}
	found := false
	for i := range providers {
		if providers[i].Name == name {
			providers[i].Enabled = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown agent %q", name)
	}
	if err := store.Save(providers); err != nil {
		return err
	}
	if err := selection.Save(agents.Selection{Active: name}); err != nil {
		return err
	}
	return h.emitAgentsList(ctx, action)
}

func (h SessionHandler) toggleAgent(ctx context.Context, action protocol.Action, name string, enabled bool) error {
	if name == "" {
		return fmt.Errorf("agent name required")
	}
	store, selection, err := h.agentStores()
	if err != nil {
		return err
	}
	providers, err := store.Load()
	if err != nil {
		return err
	}
	found := false
	for i := range providers {
		if providers[i].Name == name {
			providers[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown agent %q", name)
	}
	if err := store.Save(providers); err != nil {
		return err
	}
	sel, err := selection.Load()
	if err != nil {
		return err
	}
	if !enabled && sel.Active == name {
		sel.Active = ""
		if err := selection.Save(sel); err != nil {
			return err
		}
	}
	return h.emitAgentsList(ctx, action)
}

func (h SessionHandler) loadAgents(ctx context.Context) ([]protocol.AgentProviderInfo, string, error) {
	store, selection, err := h.agentStores()
	if err != nil {
		return nil, "", err
	}
	providers, err := store.Load()
	if err != nil {
		return nil, "", err
	}
	sel, err := selection.Load()
	if err != nil {
		return nil, "", err
	}
	out := make([]protocol.AgentProviderInfo, 0, len(providers))
	for _, prov := range providers {
		hasKey := h.providerHasKey(prov)
		out = append(out, protocol.AgentProviderInfo{
			Name:      prov.Name,
			Type:      string(prov.Type),
			Model:     prov.Model,
			BaseURL:   prov.BaseURL,
			APIKeyEnv: prov.APIKeyEnv,
			HasAPIKey: hasKey,
			Enabled:   prov.Enabled,
			Status:    providerStatus(ctx, prov, h.Session.agentsChecker, hasKey),
		})
	}
	return out, sel.Active, nil
}

func (h SessionHandler) agentStores() (AgentsStore, AgentsSelectionStore, error) {
	if h.Session == nil || h.Session.agentsStore == nil || h.Session.agentsSelection == nil {
		return nil, nil, fmt.Errorf("agents store not configured")
	}
	return h.Session.agentsStore, h.Session.agentsSelection, nil
}

func (h SessionHandler) handleAgentPrompt(ctx context.Context, action protocol.Action, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("prompt content required")
	}
	if h.Session == nil || h.Session.agentsRunner == nil {
		return h.emitAgentMessage(action, "No agent runner configured.")
	}
	provider, err := h.resolveActiveProvider()
	if err != nil {
		return h.emitAgentMessage(action, err.Error())
	}
	if h.Session != nil && h.Session.agentsSecrets != nil {
		if key, err := h.Session.agentsSecrets.Get(provider.Name); err == nil && key != "" {
			provider.APIKey = key
		}
	}
	resp, err := h.Session.agentsRunner.Prompt(ctx, provider, content)
	if err != nil {
		return h.emitAgentMessage(action, fmt.Sprintf("Agent error: %v", err))
	}
	if strings.TrimSpace(resp) == "" {
		return h.emitAgentMessage(action, "Agent returned empty response.")
	}
	return h.emitAgentMessage(action, resp)
}

func (h SessionHandler) emitAgentMessage(action protocol.Action, content string) error {
	payload := protocol.AgentMessagePayload{Content: content}
	data, _ := protocol.MarshalPayload(payload)
	h.Session.Emit(protocol.Event{
		EventID:   "agent-message",
		SessionID: action.SessionID,
		Timestamp: time.Now().UTC(),
		Type:      protocol.EventAgentMessage,
		Source:    protocol.SourceAgent,
		AgentID:   action.AgentID,
		Payload:   data,
	})
	return nil
}

func (h SessionHandler) resolveActiveProvider() (agents.Provider, error) {
	store, selection, err := h.agentStores()
	if err != nil {
		return agents.Provider{}, err
	}
	providers, err := store.Load()
	if err != nil {
		return agents.Provider{}, err
	}
	sel, err := selection.Load()
	if err != nil {
		return agents.Provider{}, err
	}
	if sel.Active != "" {
		for _, prov := range providers {
			if prov.Name == sel.Active {
				if !prov.Enabled {
					return agents.Provider{}, fmt.Errorf("active provider %q is disabled", prov.Name)
				}
				return prov, nil
			}
		}
		return agents.Provider{}, fmt.Errorf("active provider %q not found", sel.Active)
	}
	for _, prov := range providers {
		if prov.Enabled {
			return prov, nil
		}
	}
	return agents.Provider{}, fmt.Errorf("no enabled providers configured")
}

func (h SessionHandler) addAgent(ctx context.Context, action protocol.Action, payload protocol.AgentsAddPayload) error {
	if payload.Name == "" || payload.Type == "" || payload.Model == "" || (payload.APIKey == "" && payload.APIKeyEnv == "") {
		return fmt.Errorf("name, type, model, and api key (or env) are required")
	}
	pt := agents.ProviderType(strings.ToLower(payload.Type))
	if pt != agents.ProviderOpenAI &&
		pt != agents.ProviderAnthropic &&
		pt != agents.ProviderGrok &&
		pt != agents.ProviderGemini &&
		pt != agents.ProviderLLaMA {
		return fmt.Errorf("invalid provider type")
	}
	store, _, err := h.agentStores()
	if err != nil {
		return err
	}
	prov := agents.Provider{
		Name:      payload.Name,
		Type:      pt,
		Model:     payload.Model,
		BaseURL:   payload.BaseURL,
		APIKeyEnv: payload.APIKeyEnv,
		Enabled:   payload.Enabled,
	}
	if payload.APIKey != "" {
		if err := h.storeSecret(payload.Name, payload.APIKey); err != nil {
			return err
		}
	}
	if err := store.Save(upsertProviders(store, prov)); err != nil {
		return err
	}
	return h.emitAgentsList(ctx, action)
}

func (h SessionHandler) providerHasKey(prov agents.Provider) bool {
	if h.Session == nil || h.Session.agentsSecrets == nil {
		return false
	}
	_, err := h.Session.agentsSecrets.Get(prov.Name)
	return err == nil
}

func (h SessionHandler) storeSecret(name, secret string) error {
	if h.Session == nil || h.Session.agentsSecrets == nil {
		return fmt.Errorf("keychain not configured")
	}
	return h.Session.agentsSecrets.Set(name, secret)
}

func (h SessionHandler) deleteAgent(ctx context.Context, action protocol.Action, name string) error {
	if name == "" {
		return fmt.Errorf("agent name required")
	}
	store, selection, err := h.agentStores()
	if err != nil {
		return err
	}
	providers, err := store.Load()
	if err != nil {
		return err
	}
	found := false
	out := providers[:0]
	for _, prov := range providers {
		if prov.Name == name {
			found = true
			continue
		}
		out = append(out, prov)
	}
	if !found {
		return fmt.Errorf("unknown agent %q", name)
	}
	if err := store.Save(out); err != nil {
		return err
	}
	if h.Session != nil && h.Session.agentsSecrets != nil {
		_ = h.Session.agentsSecrets.Delete(name)
	}
	sel, err := selection.Load()
	if err != nil {
		return err
	}
	if sel.Active == name {
		sel.Active = ""
		if err := selection.Save(sel); err != nil {
			return err
		}
	}
	return h.emitAgentsList(ctx, action)
}

func providerStatus(ctx context.Context, prov agents.Provider, checker AgentsChecker, hasKey bool) string {
	if !prov.Enabled {
		return "disabled"
	}
	if prov.Type == agents.ProviderLLaMA && prov.BaseURL == "" {
		return "missing_base_url"
	}
	if !hasKey {
		if prov.APIKeyEnv != "" {
			if _, ok := os.LookupEnv(prov.APIKeyEnv); !ok {
				return "missing_key"
			}
		} else {
			return "missing_key"
		}
	}
	if checker != nil {
		status := checker.Check(ctx, prov)
		if status != "" {
			return status
		}
	}
	return "ready"
}

func upsertProviders(store AgentsStore, prov agents.Provider) []agents.Provider {
	providers, err := store.Load()
	if err != nil {
		return []agents.Provider{prov}
	}
	updated := false
	for i := range providers {
		if providers[i].Name == prov.Name {
			providers[i] = prov
			updated = true
			break
		}
	}
	if !updated {
		providers = append(providers, prov)
	}
	return providers
}

func toProtocolContainers(items []containers.CapsuleInfo) []protocol.ContainerInfo {
	out := make([]protocol.ContainerInfo, 0, len(items))
	for _, item := range items {
		out = append(out, protocol.ContainerInfo{
			ID:     item.ID,
			Name:   item.Name,
			RepoID: item.RepoID,
			Labels: item.Labels,
		})
	}
	return out
}
