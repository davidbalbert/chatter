package sync

import "context"

type Token struct {
	t chan struct{}
}

// A struct that facilitates one-to-many broadcast notifications. All listeners are guaranteed
// to be notified of every change, and once you've registered, you won't miss any changes.
//
// St is a buffered channel that acts as a mutex for the notifier's state. The state is a map
// from arbitrary unique values (in this case a channelvalue ) to queues. The queues function
// similarly to a buffered channel, except they  can grow infinitely, which means a slow listener
// will never block the notifier. Essentially, you're trading potentially unbounded memory growth
// for the guarantee that you won't miss any messages and no goroutine can slow another down.
//
// The channel is wrapped in a Token struct to hide its implementation details. No values are
// ever sent on the channel, it's just used a unique value.
//
// One thing this is missing right now: when you register a listener, it's sometimes useful to
// immediately receive the most recent value. Right now, you can't do that, but it would be
// pretty easy to add "lastValue" to the state.
type QueuedNotifier[T any] struct {
	st chan map[chan struct{}]*queue[T]
}

func NewQueuedNotifier[T any]() *QueuedNotifier[T] {
	state := make(chan map[chan struct{}]*queue[T], 1)
	state <- make(map[chan struct{}]*queue[T])

	return &QueuedNotifier[T]{
		st: state,
	}
}

func (n *QueuedNotifier[T]) Register() Token {
	q := NewQueue[T]()
	t := make(chan struct{})

	st := <-n.st
	st[t] = q
	n.st <- st

	return Token{t}
}

func (n *QueuedNotifier[T]) Unregister(t Token) {
	st := <-n.st
	delete(st, t.t)
	n.st <- st
}

func (n *QueuedNotifier[T]) NotifyChange(v T) {
	st := <-n.st
	for _, q := range st {
		q.Put(v)
	}
	n.st <- st
}

func (n *QueuedNotifier[T]) AwaitChange(ctx context.Context, t Token) (T, bool) {
	st := <-n.st
	q := st[t.t]
	n.st <- st

	if q == nil {
		var zero T
		return zero, false
	}

	return q.Get(ctx), true
}
