package system

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/davidbalbert/chatter/chatterd/services"
	"golang.org/x/sync/errgroup"
)

type macosInterfaceMonitor struct {
	*baseInterfaceMonitor
}

func NewInterfaceMonitor(serviceManager *services.ServiceManager, conf any) (services.Service, error) {
	base := newBaseInterfaceMonitor()

	return &macosInterfaceMonitor{
		baseInterfaceMonitor: base,
	}, nil
}

func (m *macosInterfaceMonitor) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// We want scutil to exit cleanly when ctx is canceled, which we can
	// achieve by closing stdin. We're not using exec.CommandContext
	// because it causes scutil to exit with an error.
	cmd := exec.Command("scutil")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get scutil stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get scutil stdout pipe: %w", err)
	}

	events := make(chan struct{})

	g.Go(func() error {
		defer stdin.Close()

		// TODO: is there a way to get a notification when an interface
		// goes from loopback to not loopback?
		input := `
			n.add State:/Network/Interface
			n.add State:/Network/Interface/[^/]+/Link "pattern"
			n.add State:/Network/Interface/[^/]+/IPv4 "pattern"
			n.add State:/Network/Interface/[^/]+/IPv6 "pattern"
			n.watch
		`

		_, err := io.WriteString(stdin, input)
		if err != nil {
			return fmt.Errorf("failed to write to scutil stdin: %w", err)
		}

		<-ctx.Done()

		return nil
	})

	g.Go(func() error {
		r := bufio.NewReader(stdout)

		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				return nil
			} else if err != nil {
				return fmt.Errorf("failed to read from scutil stdout: %w", err)
			}

			if strings.Contains(line, "State:/Network/Interface/") {
				select {
				case events <- struct{}{}:
				default:
				}
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
			pending     bool
		)

		for {
			select {
			case <-ctx.Done():
				if notifyTimer != nil && !notifyTimer.Stop() {
					<-notifyTimer.C
				}

				if pending {
					m.notify()
				}

				return nil
			case <-events:
				pending = true

				if notifyTimer == nil {
					notifyTimer = time.NewTimer(200 * time.Millisecond)
					notifyCh = notifyTimer.C
				}
			case <-notifyCh:
				m.notify()
				pending = false
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
