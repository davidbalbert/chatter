package sync

import (
	"context"
)

// Adapted from the slides for "Rethinking Classical Concurrency Patterns" by Bryan C. Mills.

type queue[T any] struct {
	items chan []T  // contains 0 or 1 non-empty slices
	empty chan bool // contains true if items is empty
}

func NewQueue[T any]() *queue[T] {
	items := make(chan []T, 1)
	empty := make(chan bool, 1)
	empty <- true
	return &queue[T]{items, empty}
}

func (q *queue[T]) Put(item T) {
	var items []T
	select {
	case items = <-q.items:
	case <-q.empty:
	}
	items = append(items, item)
	q.items <- items
}

func (q *queue[T]) Get(ctx context.Context) T {
	var items []T
	select {
	case <-ctx.Done():
		var zero T
		return zero
	case items = <-q.items:
	}

	item := items[0]
	items = items[1:]
	if len(items) == 0 {
		q.empty <- true
	} else {
		q.items <- items
	}

	return item
}
