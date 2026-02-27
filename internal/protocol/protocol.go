package protocol

import (
	"encoding/json"
	"errors"
	"time"
)

const ProtocolVersion = 1

type ActionType string

const (
	ActionRunCommand    ActionType = "run_command"
	ActionApplyPatch    ActionType = "apply_patch"
	ActionFreeze        ActionType = "freeze"
	ActionReset         ActionType = "reset"
	ActionNetworkToggle ActionType = "network_toggle"
	ActionContainersList ActionType = "containers_list"
	ActionRevert        ActionType = "revert"
	ActionAgentsList    ActionType = "agents_list"
)

type Action struct {
	ActionID  string          `json:"action_id"`
	SessionID string          `json:"session_id"`
	AgentID   string          `json:"agent_id"`
	Type      ActionType      `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type EventType string

type EventSource string

const (
	SourceSystem   EventSource = "system"
	SourceExecutor EventSource = "executor"
	SourceAgent    EventSource = "agent"
)

const (
	EventSessionStarted EventType = "session.started"
	EventExecutorBusy   EventType = "executor.busy"
	EventExecutorIdle   EventType = "executor.idle"
	EventActionStarted  EventType = "action.started"
	EventActionFinished EventType = "action.finished"
	EventTerminalOutput EventType = "terminal.output"
	EventAgentMessage   EventType = "agent.message"
	EventDiffReady      EventType = "diff.ready"
	EventFreezeCreated  EventType = "freeze.created"
	EventResetCompleted EventType = "reset.completed"
	EventNetworkChanged EventType = "network.changed"
	EventContainersList EventType = "containers.list"
	EventError          EventType = "error"
)

type Event struct {
	EventID   string          `json:"event_id"`
	SessionID string          `json:"session_id"`
	Timestamp time.Time       `json:"timestamp"`
	Type      EventType       `json:"type"`
	Source    EventSource     `json:"source"`
	AgentID   string          `json:"agent_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// Action payloads.
type RunCommandPayload struct {
	Command string            `json:"command"`
	Cwd     string            `json:"cwd"`
	Env     map[string]string `json:"env,omitempty"`
}

type ApplyPatchPayload struct {
	Patch string `json:"patch"`
}

type FreezePayload struct {
	Mode string `json:"mode"`
}

type ResetPayload struct {
	PreserveVolumes bool `json:"preserve_volumes"`
}

type NetworkTogglePayload struct {
	Enabled bool `json:"enabled"`
}

// Event payloads.
type SessionStartedPayload struct {
	RepoRoot    string `json:"repo_root"`
	CapsuleName string `json:"capsule_name"`
}

type ExecutorBusyPayload struct {
	ActionID string `json:"action_id"`
}

type ActionStartedPayload struct {
	ActionID string `json:"action_id"`
	Type     string `json:"type"`
}

type ActionFinishedPayload struct {
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type TerminalOutputPayload struct {
	Stream string `json:"stream"`
	Data   string `json:"data"`
}

type AgentMessagePayload struct {
	Content string `json:"content"`
}

type DiffReadyPayload struct {
	Patch string   `json:"patch"`
	Files []string `json:"files"`
}

type FreezeCreatedPayload struct {
	Image     string `json:"image"`
	SizeBytes int64  `json:"size_bytes"`
}

type NetworkChangedPayload struct {
	Enabled bool `json:"enabled"`
}

type ContainersListPayload struct {
	Capsules []ContainerInfo `json:"capsules"`
}

type ContainerInfo struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	RepoID string            `json:"repo_id"`
	Labels map[string]string `json:"labels"`
}

type ErrorPayload struct {
	Message  string `json:"message"`
	ActionID string `json:"action_id,omitempty"`
}

var validActionTypes = map[ActionType]struct{}{
	ActionRunCommand:    {},
	ActionApplyPatch:    {},
	ActionFreeze:        {},
	ActionReset:         {},
	ActionNetworkToggle: {},
	ActionContainersList: {},
	ActionRevert:        {},
	ActionAgentsList:    {},
}

var validEventTypes = map[EventType]struct{}{
	EventSessionStarted: {},
	EventExecutorBusy:   {},
	EventExecutorIdle:   {},
	EventActionStarted:  {},
	EventActionFinished: {},
	EventTerminalOutput: {},
	EventAgentMessage:   {},
	EventDiffReady:      {},
	EventFreezeCreated:  {},
	EventResetCompleted: {},
	EventNetworkChanged: {},
	EventContainersList: {},
	EventError:          {},
}

var validEventSources = map[EventSource]struct{}{
	SourceSystem:   {},
	SourceExecutor: {},
	SourceAgent:    {},
}

func ValidateAction(action Action) error {
	if action.ActionID == "" {
		return errors.New("action_id is required")
	}
	if action.SessionID == "" {
		return errors.New("session_id is required")
	}
	if action.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if action.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	if _, ok := validActionTypes[action.Type]; !ok {
		return errors.New("unknown action type")
	}
	return nil
}

func ValidateEvent(event Event) error {
	if event.EventID == "" {
		return errors.New("event_id is required")
	}
	if event.SessionID == "" {
		return errors.New("session_id is required")
	}
	if event.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	if _, ok := validEventTypes[event.Type]; !ok {
		return errors.New("unknown event type")
	}
	if _, ok := validEventSources[event.Source]; !ok {
		return errors.New("unknown event source")
	}
	return nil
}

// MarshalPayload encodes payloads for Action/Event messages.
func MarshalPayload(payload any) ([]byte, error) {
	return json.Marshal(payload)
}
