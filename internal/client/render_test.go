package client

import (
	"strings"
	"testing"
	"time"

	"krellin/internal/protocol"
)

func TestFormatEvent(t *testing.T) {
	ev := protocol.Event{
		EventID:   "e1",
		SessionID: "s1",
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		Type:      protocol.EventActionStarted,
		Source:    protocol.SourceExecutor,
		AgentID:   "agent",
	}
	line := FormatEvent(ev)
	if !strings.Contains(line, "action.started") {
		t.Fatalf("expected event type in line: %q", line)
	}
	if !strings.Contains(line, "agent") {
		t.Fatalf("expected agent in line: %q", line)
	}
}
