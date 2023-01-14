package vty

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

const socketPath = "/tmp/ospfd.sock"

type framedConn struct {
	net.Conn
}

func (c *framedConn) ReadFrame() ([]byte, error) {
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

func (c *framedConn) WriteFrame(b []byte) error {
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
	done     chan struct{}
	handler  func(io.Writer, string)
}

func NewServer() *Server {
	return &Server{
		done: make(chan struct{}),
	}
}

func (s *Server) HandleFunc(f func(io.Writer, string)) {
	s.handler = f
}

func (s *Server) handle(conn *framedConn) error {
	defer conn.Close()

	for {
		frame, err := conn.ReadFrame()
		if err != nil {
			select {
			case <-s.done:
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

		if err := conn.WriteFrame(b.Bytes()); err != nil {
			return err
		}
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	s.listener = listener

	for {
		select {
		case <-ctx.Done():
			g.Wait()
			return ctx.Err()
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				g.Wait()
				return nil
			default:
				g.Wait()
				return err
			}
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		g.Go(func() error {
			return s.handle(&framedConn{conn})
		})
	}
}

func (s *Server) Shutdown() error {
	close(s.done)
	ferr := s.listener.Close()

	s.mu.Lock()
	for _, conn := range s.conns {
		fmt.Printf("closing conn: %v\n", conn)
		err := conn.Close()
		if err != nil {
			ferr = err
		}
	}
	s.mu.Unlock()

	return ferr
}

type Client struct {
	conn framedConn
}

func Dial() (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn: framedConn{conn},
	}, nil
}

func (c *Client) Send(cmd string) (string, error) {
	if err := c.conn.WriteFrame([]byte(cmd)); err != nil {
		return "", err
	}

	frame, err := c.conn.ReadFrame()
	if err != nil {
		return "", err
	}

	return string(frame), nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
