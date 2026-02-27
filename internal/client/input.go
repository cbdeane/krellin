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

type Input struct {
	client client.Client
	in     io.Reader
	out    io.Writer
}

func NewInput(c client.Client, in io.Reader, out io.Writer) *Input {
	return &Input{client: c, in: in, out: out}
}

func (i *Input) Run(ctx context.Context, sessionID string, agentID string) error {
	scanner := bufio.NewScanner(i.in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if i.out != nil {
			_, _ = i.out.Write([]byte("> " + line + "\n"))
		}
		if strings.HasPrefix(line, "/agents") {
			action := protocol.Action{
				ActionID:  "local",
				SessionID: sessionID,
				AgentID:   agentID,
				Type:      protocol.ActionAgentsList,
				Timestamp: time.Now(),
				Payload:   encodeJSON(struct{}{}),
			}
			if err := i.client.SendAction(ctx, encodeJSON(action)); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "!") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
			if line == "" {
				continue
			}
		}
		action := protocol.Action{
			ActionID:  "local",
			SessionID: sessionID,
			AgentID:   agentID,
			Type:      protocol.ActionRunCommand,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.RunCommandPayload{Command: line, Cwd: "/workspace"}),
		}
		if err := i.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func encodeJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
