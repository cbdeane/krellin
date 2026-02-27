package executor

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"krellin/internal/protocol"
	"krellin/internal/queue"
)

// Handler executes a single action.
type Handler interface {
	Handle(ctx context.Context, action protocol.Action) error
}

// Emitter publishes events from the executor.
type Emitter interface {
	Emit(event protocol.Event)
}

type Executor struct {
	queue   *queue.Queue[protocol.Action]
	handler Handler
	emitter Emitter
}

func New(q *queue.Queue[protocol.Action], handler Handler, emitter Emitter) *Executor {
	return &Executor{queue: q, handler: handler, emitter: emitter}
}

func (e *Executor) Run(ctx context.Context) {
	for {
		action, err := e.queue.Dequeue(ctx)
		if err != nil {
			return
		}

		e.emit(action, protocol.EventExecutorBusy, protocol.SourceExecutor, protocol.ExecutorBusyPayload{ActionID: action.ActionID})
		e.emit(action, protocol.EventActionStarted, protocol.SourceExecutor, protocol.ActionStartedPayload{ActionID: action.ActionID, Type: string(action.Type)})

		err = e.handler.Handle(ctx, action)
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failure"
			errMsg = err.Error()
		}
		e.emit(action, protocol.EventActionFinished, protocol.SourceExecutor, protocol.ActionFinishedPayload{ActionID: action.ActionID, Status: status, Error: errMsg})
		e.emit(action, protocol.EventExecutorIdle, protocol.SourceExecutor, struct{}{})
	}
}

var eventCounter uint64

func (e *Executor) emit(action protocol.Action, eventType protocol.EventType, source protocol.EventSource, payload any) {
	if e.emitter == nil {
		return
	}
	data := protocolMarshal(payload)
	id := atomic.AddUint64(&eventCounter, 1)
	e.emitter.Emit(protocol.Event{
		EventID:   fmt.Sprintf("e-%d", id),
		SessionID: action.SessionID,
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Source:    source,
		AgentID:   action.AgentID,
		Payload:   data,
	})
}

func protocolMarshal(payload any) []byte {
	data, err := protocol.MarshalPayload(payload)
	if err != nil {
		return nil
	}
	return data
}
