package client

import (
	"context"
	"encoding/json"
	"io"
	"time"

	tea "charm.land/bubbletea/v2"
	"krellin/internal/protocol"
	"krellin/pkg/client"
)

// TUI renders the interactive terminal UI using Bubble Tea.
type TUI struct {
	client    client.Client
	sessionID string
	agentID   string
	in        io.Reader
	out       io.Writer
}

func NewTUI(c client.Client, out io.Writer, in io.Reader, sessionID string, agentID string) *TUI {
	return &TUI{
		client:    c,
		sessionID: sessionID,
		agentID:   agentID,
		in:        in,
		out:       out,
	}
}

func (t *TUI) Run(ctx context.Context) error {
	model := newTUIModel(t.client, t.sessionID, t.agentID)
	opts := []tea.ProgramOption{}
	if t.in != nil {
		opts = append(opts, tea.WithInput(t.in))
	}
	if t.out != nil {
		opts = append(opts, tea.WithOutput(t.out))
	}
	program := tea.NewProgram(model, opts...)

	go func() {
		<-ctx.Done()
		program.Send(tea.Quit())
	}()

	go func() {
		for {
			events, err := t.client.Subscribe(ctx)
			if err != nil {
				program.Send(errMsg{err: err})
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}
			program.Send(connectedMsg{})
			for msg := range events {
				var ev protocol.Event
				if err := json.Unmarshal(msg, &ev); err != nil {
					continue
				}
				program.Send(eventMsg{ev: ev})
			}
			program.Send(disconnectedMsg{})
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}()

	_, err := program.Run()
	return err
}
