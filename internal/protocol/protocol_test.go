package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestActionJSONRoundTrip(t *testing.T) {
	payload := RunCommandPayload{
		Command: "echo hi",
		Cwd:     "/workspace",
		Env:     map[string]string{"FOO": "bar"},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	in := Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      ActionRunCommand,
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		Payload:   payloadBytes,
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal action: %v", err)
	}

	var out Action
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}

	if out.ActionID != in.ActionID || out.Type != in.Type || string(out.Payload) != string(in.Payload) {
		t.Fatalf("round trip mismatch: %+v vs %+v", out, in)
	}
}

func TestActionValidate(t *testing.T) {
	base := Action{
		ActionID:  "a1",
		SessionID: "s1",
		AgentID:   "agent",
		Type:      ActionRunCommand,
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		Payload:   json.RawMessage(`{"command":"echo"}`),
	}

	if err := ValidateAction(base); err != nil {
		t.Fatalf("valid action rejected: %v", err)
	}

	missing := base
	missing.ActionID = ""
	if err := ValidateAction(missing); err == nil {
		t.Fatalf("expected error for missing action_id")
	}

	unknown := base
	unknown.Type = "nope"
	if err := ValidateAction(unknown); err == nil {
		t.Fatalf("expected error for unknown type")
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	payload := TerminalOutputPayload{Stream: "stdout", Data: "ok"}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	in := Event{
		EventID:   "e1",
		SessionID: "s1",
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		Type:      EventTerminalOutput,
		Source:    SourceExecutor,
		AgentID:   "agent",
		Payload:   payloadBytes,
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var out Event
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if out.EventID != in.EventID || out.Type != in.Type || string(out.Payload) != string(in.Payload) {
		t.Fatalf("round trip mismatch: %+v vs %+v", out, in)
	}
}

func TestEventValidate(t *testing.T) {
	base := Event{
		EventID:   "e1",
		SessionID: "s1",
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		Type:      EventSessionStarted,
		Source:    SourceSystem,
		AgentID:   "",
		Payload:   json.RawMessage(`{"repo_root":"/tmp","capsule_name":"krellin-1"}`),
	}

	if err := ValidateEvent(base); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	missing := base
	missing.EventID = ""
	if err := ValidateEvent(missing); err == nil {
		t.Fatalf("expected error for missing event_id")
	}

	unknown := base
	unknown.Type = "mystery"
	if err := ValidateEvent(unknown); err == nil {
		t.Fatalf("expected error for unknown event type")
	}
}
