package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrClosed = errors.New("queue closed")

// Queue is a FIFO blocking queue.
type Queue[T any] struct {
	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
	items  []T
}

func New[T any]() *Queue[T] {
	q := &Queue[T]{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *Queue[T]) Enqueue(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.items = append(q.items, item)
	q.cond.Signal()
	return nil
}

func (q *Queue[T]) Dequeue(ctx context.Context) (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	done := make(chan struct{})
	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				q.mu.Lock()
				q.cond.Broadcast()
				q.mu.Unlock()
			case <-done:
			}
		}()
	}
	defer close(done)

	for {
		if q.closed {
			return zero, ErrClosed
		}
		if len(q.items) > 0 {
			item := q.items[0]
			q.items = q.items[1:]
			return item, nil
		}
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		q.cond.Wait()
	}
}

func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}
