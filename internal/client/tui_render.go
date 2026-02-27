package client

import (
	"strings"

	"krellin/internal/protocol"
)

type Timeline struct {
	lines []string
	max   int
}

func NewTimeline(max int) *Timeline {
	return &Timeline{max: max}
}

func (t *Timeline) Add(ev protocol.Event) {
	line := FormatEvent(ev)
	t.lines = append(t.lines, line)
	if len(t.lines) > t.max {
		t.lines = t.lines[len(t.lines)-t.max:]
	}
}

func (t *Timeline) Render() string {
	return strings.Join(t.lines, "\n")
}

// FormatEvent is shared in render.go.
