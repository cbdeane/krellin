package client

import (
	"fmt"
	"time"

	"krellin/internal/protocol"
)

// FormatEvent renders a single event line for the TUI.
func FormatEvent(ev protocol.Event) string {
	ts := ev.Timestamp.UTC().Format(time.RFC3339)
	if ev.AgentID != "" {
		return fmt.Sprintf("%s [%s] %s (%s)", ts, ev.Source, ev.Type, ev.AgentID)
	}
	return fmt.Sprintf("%s [%s] %s", ts, ev.Source, ev.Type)
}
