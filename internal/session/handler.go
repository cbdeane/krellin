package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		if _, err := s.pty.Write([]byte(cmd + "\n")); err != nil {
			return err
		}
		// Echo command to terminal to improve visibility even if PTY output is delayed.
		echo, _ := protocol.MarshalPayload(protocol.TerminalOutputPayload{
			Stream: "stdout",
			Data:   "> " + cmd + "\n",
		})
		s.Emit(protocol.Event{
			EventID:   "command-echo",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventTerminalOutput,
			Source:    protocol.SourceExecutor,
			AgentID:   action.AgentID,
			Payload:   echo,
		})
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
		payload := protocol.AgentMessagePayload{Content: "No agents connected (LLM backend not configured)."}
		data, _ := protocol.MarshalPayload(payload)
		s.Emit(protocol.Event{
			EventID:   "agent-message",
			SessionID: action.SessionID,
			Timestamp: time.Now().UTC(),
			Type:      protocol.EventAgentMessage,
			Source:    protocol.SourceSystem,
			AgentID:   action.AgentID,
			Payload:   data,
		})
		return nil
	default:
		return fmt.Errorf("action not implemented")
	}
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
