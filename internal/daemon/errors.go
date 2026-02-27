package daemon

import (
	"time"

	"krellin/internal/protocol"
)

func errorEvent(sessionID string, actionID string, msg string) protocol.Event {
	payload, _ := protocol.MarshalPayload(protocol.ErrorPayload{Message: msg, ActionID: actionID})
	return protocol.Event{
		EventID:   "error",
		SessionID: sessionID,
		Timestamp: time.Now().UTC(),
		Type:      protocol.EventError,
		Source:    protocol.SourceSystem,
		Payload:   payload,
	}
}
