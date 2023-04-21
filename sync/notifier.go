package sync

import "context"

type state struct {
	seq     int64
	changed chan struct{} // closed upon notify
}

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

// Calling Seq is usually unnecessary. You can just start with 0, and AwaitChange
// will give you the correct seq. But if getting signaled twice is expensive, you
// can limit the likelyhood of getting signaled twice by calling Seq() first.
func (n *Notifier) Seq() int64 {
	st := <-n.st
	n.st <- st
	return st.seq
}
