package system

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type macosInterfaceMonitor struct {
	baseInterfaceMonitor
}

func NewInterfaceMonitor() InterfaceMonitor {
	return &macosInterfaceMonitor{}
}

func (m *macosInterfaceMonitor) readEvent(r *bufio.Reader) (*InterfaceEvent, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if !strings.Contains(line, "changedKey") {
		return nil, nil
	}

	trimmed := strings.TrimSpace(line)

	i := strings.LastIndex(trimmed, "State:/Network/Interface/")
	if i == -1 {
		return nil, nil
	}

	key := trimmed[i:]

	return &InterfaceEvent{detail: key}, nil
}

func (m *macosInterfaceMonitor) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// We want scutil to exit cleanly when ctx is canceled, which we can
	// achieve by closing stdin. We're not using exec.CommandContext
	// because it causes scutil to exit with an error.
	cmd := exec.Command("scutil")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	eventCh := make(chan *InterfaceEvent)

	g.Go(func() error {
		defer stdin.Close()

		input := `
			n.add State:/Network/Interface
			n.add State:/Network/Interface/[^/]+/Link "pattern"
			n.add State:/Network/Interface/[^/]+/IPv4 "pattern"
			n.add State:/Network/Interface/[^/]+/IPv6 "pattern"
			n.watch
		`

		_, err := io.WriteString(stdin, input)
		if err != nil {
			return err
		}

		<-ctx.Done()

		close(eventCh)

		return nil
	})

	g.Go(func() error {
		r := bufio.NewReader(stdout)

		for {
			event, err := m.readEvent(r)
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}

			if event != nil {
				eventCh <- event
			}
		}
	})

	g.Go(func() error {
		// We want to batch events together, so we wait after
		// receiving an event for other events to accumulate before
		// notifying listeners.

		var (
			notifyTimer *time.Timer
			notifyCh    <-chan time.Time
			events      []InterfaceEvent
		)

		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					if notifyTimer != nil && !notifyTimer.Stop() {
						<-notifyTimer.C
					}

					if len(events) > 0 {
						m.notify(events)
					}

					return nil
				}

				events = append(events, *event)

				if notifyTimer == nil {
					notifyTimer = time.NewTimer(200 * time.Millisecond)
					notifyCh = notifyTimer.C
				}
			case <-notifyCh:
				m.notify(events)
				events = nil
				notifyCh = nil
				notifyTimer = nil
			}
		}
	})

	g.Go(func() error {
		return cmd.Run()
	})

	return g.Wait()
}
