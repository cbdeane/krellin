package client

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"krellin/internal/protocol"
	"krellin/pkg/client"
)

// TUI is a minimal event renderer with timeline, terminal, and diff panes.
type TUI struct {
	client    client.Client
	out       io.Writer
	timeline  *Timeline
	terminal  []string
	diff      []string
	input     *Input
	sessionID string
	agentID   string
	mode      ViewMode
}

func NewTUI(c client.Client, out io.Writer, in io.Reader, sessionID string, agentID string) *TUI {
	return &TUI{
		client:    c,
		out:       out,
		timeline:  NewTimeline(8),
		input:     NewInput(c, in, out),
		sessionID: sessionID,
		agentID:   agentID,
		mode:      ViewAll,
	}
}

func (t *TUI) Run(ctx context.Context) error {
	w := bufio.NewWriter(t.out)
	// Initial render so the user sees the UI immediately.
	_, _ = w.WriteString(t.render())
	_ = w.Flush()

	go func() {
		_ = t.input.Run(ctx, t.sessionID, t.agentID)
	}()
	for {
		events, err := t.client.Subscribe(ctx)
		if err != nil {
			t.terminal = clampLines(append(t.terminal, "[error] "+err.Error()+"\n"), 10)
			_, _ = w.WriteString(t.render())
			_ = w.Flush()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
				continue
			}
		}

		for {
			select {
			case <-ctx.Done():
				_ = w.Flush()
				return ctx.Err()
			case msg, ok := <-events:
				if !ok {
					_, _ = w.WriteString("[disconnected] retrying...\n")
					_ = w.Flush()
					time.Sleep(500 * time.Millisecond)
					goto reconnect
				}
				var ev protocol.Event
				if err := json.Unmarshal(msg, &ev); err != nil {
					continue
				}
				t.applyEvent(ev)
				_, _ = w.WriteString(t.render())
				_ = w.Flush()
			}
		}
	reconnect:
		continue
	}
}

func (t *TUI) applyEvent(ev protocol.Event) {
	t.timeline.Add(ev)
	switch ev.Type {
	case protocol.EventTerminalOutput:
		var payload protocol.TerminalOutputPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			t.terminal = clampLines(append(t.terminal, payload.Data), 10)
			t.timeline.Add(protocol.Event{Type: protocol.EventTerminalOutput, Timestamp: time.Now(), Source: protocol.SourceExecutor, AgentID: ev.AgentID})
		}
	case protocol.EventDiffReady:
		var payload protocol.DiffReadyPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			t.diff = clampLines(strings.Split(payload.Patch, "\n"), 12)
		}
	case protocol.EventError:
		var payload protocol.ErrorPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			t.timeline.Add(ev)
			t.terminal = clampLines(append(t.terminal, "[error] "+payload.Message+"\n"), 10)
		}
	case protocol.EventActionFinished:
		var payload protocol.ActionFinishedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil && payload.Status == "failure" {
			t.terminal = clampLines(append(t.terminal, "[error] action failed: "+payload.Error+"\n"), 10)
		}
	case protocol.EventAgentMessage:
		var payload protocol.AgentMessagePayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			t.terminal = clampLines(append(t.terminal, "[agent] "+payload.Content+"\n"), 10)
		}
	}
}

func (t *TUI) render() string {
	var b strings.Builder
	b.WriteString("===== Krellin =====\n")
	b.WriteString("Type a command and press Enter to run it in the capsule.\n")
	if t.mode == ViewAll || t.mode == ViewTerminal {
		b.WriteString(renderSection("Timeline", t.timeline.Render()))
		b.WriteString("\n")
		b.WriteString(renderSection("Terminal", strings.Join(t.terminal, "")))
		b.WriteString("\n")
	}
	if t.mode == ViewAll || t.mode == ViewDiff {
		b.WriteString(renderSection("Diff", renderLines(t.diff)))
		b.WriteString("\n")
	}
	return b.String()
}
