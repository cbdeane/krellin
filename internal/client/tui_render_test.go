package client

import (
	"strings"
	"testing"
	"time"

	"krellin/internal/protocol"
)

func TestTimelineRender(t *testing.T) {
	tl := NewTimeline(2)
	tl.Add(protocol.Event{Type: protocol.EventExecutorBusy, Timestamp: time.Now()})
	tl.Add(protocol.Event{Type: protocol.EventExecutorIdle, Timestamp: time.Now()})
	tl.Add(protocol.Event{Type: protocol.EventActionFinished, Timestamp: time.Now()})

	out := tl.Render()
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("expected 2 lines, got: %q", out)
	}
}
