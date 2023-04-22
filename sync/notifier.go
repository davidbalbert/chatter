package sync

import "context"

// Taken from the slides for "Rethinking Classical Concurrency Patterns" by Bryan C. Mills.

type state struct {
	seq     int64
	changed chan struct{} // closed upon notify
}

// A struct that facilitates one-to-many broadcast notifications. All listeners are guaranteed
// to receive a notification, but if you spend too long between calls to AwaitChange(), you
// can miss notifications.
//
// Calling AwaitChange() with an out of date sequence number guarantees that you'll be notified
// immediately with the latest seq, but if you've missed two notifications and call AwaitChange()
// you'll only be notified once.
type Notifier struct {
	st chan state
}

func NewNotifier() *Notifier {
	st := make(chan state, 1)
	st <- state{
		seq:     0,
		changed: make(chan struct{}),
	}
	return &Notifier{st: st}
}

func (n *Notifier) NotifyChange() {
	st := <-n.st
	close(st.changed)
	n.st <- state{
		seq:     st.seq + 1,
		changed: make(chan struct{}),
	}
}

// If you call AwaitChange() with a wrong seq, it'll immediately notify you
// with the current one.
func (n *Notifier) AwaitChange(ctx context.Context, seq int64) (newSeq int64) {
	st := <-n.st
	n.st <- st

	if st.seq != seq {
		return st.seq
	}

	select {
	case <-ctx.Done():
		return seq
	case <-st.changed:
		return seq + 1
	}
}
