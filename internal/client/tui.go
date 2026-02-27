package client

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"krellin/internal/protocol"
	"krellin/pkg/client"
)

// TUI is a minimal event renderer with timeline, terminal, and diff panes.
type TUI struct {
	client   client.Client
	out      io.Writer
	timeline *Timeline
	terminal []string
	diff     []string
	input    *Input
	sessionID string
	agentID   string
	mode      ViewMode
}

func NewTUI(c client.Client, out io.Writer, in io.Reader, sessionID string, agentID string) *TUI {
	return &TUI{
		client:    c,
		out:       out,
		timeline:  NewTimeline(8),
		input:     NewInput(c, in),
		sessionID: sessionID,
		agentID:   agentID,
		mode:      ViewAll,
	}
}

func (t *TUI) Run(ctx context.Context) error {
	events, err := t.client.Subscribe(ctx)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(t.out)
	go func() {
		_ = t.input.Run(ctx, t.sessionID, t.agentID)
	}()
	for {
		select {
		case <-ctx.Done():
			_ = w.Flush()
			return ctx.Err()
		case msg, ok := <-events:
			if !ok {
				_ = w.Flush()
				return nil
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
}

func (t *TUI) applyEvent(ev protocol.Event) {
	t.timeline.Add(ev)
	switch ev.Type {
	case protocol.EventTerminalOutput:
		var payload protocol.TerminalOutputPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			t.terminal = clampLines(append(t.terminal, payload.Data), 10)
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
	}
}

func (t *TUI) render() string {
	var b strings.Builder
	b.WriteString("===== Krellin =====\n")
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
