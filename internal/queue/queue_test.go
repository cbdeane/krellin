package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

type item struct {
	ID int
}

func TestQueueFIFO(t *testing.T) {
	q := New[item]()
	q.Enqueue(item{ID: 1})
	q.Enqueue(item{ID: 2})
	q.Enqueue(item{ID: 3})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got1, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue 1: %v", err)
	}
	got2, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue 2: %v", err)
	}
	got3, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue 3: %v", err)
	}

	if got1.ID != 1 || got2.ID != 2 || got3.ID != 3 {
		t.Fatalf("unexpected order: %v %v %v", got1, got2, got3)
	}
}

func TestQueueBlocksUntilItem(t *testing.T) {
	q := New[item]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var got item
	var err error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		got, err = q.Dequeue(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	q.Enqueue(item{ID: 42})
	wg.Wait()

	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got.ID != 42 {
		t.Fatalf("expected 42, got %d", got.ID)
	}
}

func TestQueueClose(t *testing.T) {
	q := New[item]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	q.Close()
	_, err := q.Dequeue(ctx)
	if err == nil {
		t.Fatalf("expected error after close")
	}
}
