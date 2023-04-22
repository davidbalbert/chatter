package sync

import "context"

// Adapted from the slides for "Rethinking Classical Concurrency Patterns" by Bryan C. Mills.

type signal[T any] struct {
	seq int64
	val T
}

type state[T any] struct {
	seq  int64
	val  T
	wait []chan<- signal[T]
}

// A struct that facilitates one-to-many broadcast notifications. All listeners are guaranteed
// to receive a notification, but if you spend too long between calls to AwaitChange(), you
// can miss notifications.
//
// Calling AwaitChange() with an out of date sequence number guarantees that you'll be notified
// immediately with the latest seq, but if you've missed two notifications and call AwaitChange()
// you'll only be notified once.
type Notifier[T any] struct {
	st chan state[T]
}

func NewNotifier[T any]() *Notifier[T] {
	st := make(chan state[T], 1)
	st <- state[T]{seq: 0}
	return &Notifier[T]{st}
}

func (n *Notifier[T]) NotifyChange(newVal T) {
	st := <-n.st
	for _, c := range st.wait {
		c <- signal[T]{st.seq + 1, newVal}
	}
	n.st <- state[T]{st.seq + 1, newVal, nil}
}

// If you call AwaitChange() with a wrong seq, it'll immediately notify you
// with the current one.
func (n *Notifier[T]) AwaitChange(ctx context.Context, seq int64) (val T, newSeq int64) {
	c := make(chan signal[T], 1)
	st := <-n.st

	if st.seq == seq {
		st.wait = append(st.wait, c)
	} else {
		c <- signal[T]{seq: st.seq, val: st.val}
	}
	n.st <- st

	select {
	case <-ctx.Done():
		return st.val, st.seq
	case n := <-c:
		return n.val, n.seq
	}
}

// Same as Notifier, but with no value.
type SimpleNotifier struct {
	*Notifier[struct{}]
}

func NewSimpleNotifier() *SimpleNotifier {
	return &SimpleNotifier{NewNotifier[struct{}]()}
}

func (n *SimpleNotifier) NotifyChange() {
	n.Notifier.NotifyChange(struct{}{})
}

func (n *SimpleNotifier) AwaitChange(ctx context.Context, seq int64) (newSeq int64) {
	_, newSeq = n.Notifier.AwaitChange(ctx, seq)
	return
}
