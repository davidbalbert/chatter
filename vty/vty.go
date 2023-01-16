package vty

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

const socketPath = "/tmp/ospfd.sock"

type FramedConn struct {
	net.Conn
}

func (c *FramedConn) ReadFrame() ([]byte, error) {
	var length uint16
	if err := binary.Read(c.Conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	b := make([]byte, length)
	if _, err := io.ReadFull(c.Conn, b); err != nil {
		return nil, err
	}

	return b, nil
}

func (c *FramedConn) WriteFrame(b []byte) error {
	if len(b) > math.MaxUint16 {
		return errors.New("frame too large")
	}

	if err := binary.Write(c.Conn, binary.BigEndian, uint16(len(b))); err != nil {
		return err
	}

	if _, err := c.Conn.Write(b); err != nil {
		return err
	}

	return nil
}

type Server struct {
	listener net.Listener
	conns    []net.Conn
	mu       sync.Mutex
	handler  func(io.Writer, string)
}

func (s *Server) HandleFunc(f func(io.Writer, string)) {
	s.handler = f
}

func (s *Server) handle(ctx context.Context, conn *FramedConn) error {
	defer conn.Close()

	for {
		frame, err := conn.ReadFrame()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}

		var b bytes.Buffer
		s.handler(&b, string(frame))

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := conn.WriteFrame(b.Bytes()); err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-ctx.Done()
		s.listener.Close()

		s.mu.Lock()
		for _, conn := range s.conns {
			conn.Close()
		}
		s.mu.Unlock()
		return nil
	})

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				g.Wait()
				return nil
			default:
				return g.Wait()
			}
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		g.Go(func() error {
			return s.handle(ctx, &FramedConn{conn})
		})
	}
}

func Dial() (*FramedConn, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	return &FramedConn{conn}, nil
}
