package echo

import (
	"fmt"
	"net"

	"github.com/danmrichards/udpecho/internal/epoll"
	"github.com/danmrichards/udpecho/internal/sock"

	"golang.org/x/sys/unix"
)

// Server represents an echo server.
type Server struct {
	sessions map[int]string
	conn     net.PacketConn
	connsock unix.Sockaddr
	poller   *epoll.Poller
	buf      []byte
}

// NewServer returns a new echo server.
func NewServer(pc net.PacketConn, p *epoll.Poller) (*Server, error) {
	cs, err := sock.Addr(pc.LocalAddr().String())
	if err != nil {
		return nil, fmt.Errorf("server sock addr: %w", err)
	}

	return &Server{
		sessions: make(map[int]string),
		conn:     pc,
		connsock: cs,
		poller:   p,
		buf:      make([]byte, 1024),
	}, nil
}

// Close closes the server and all open sessions.
func (s *Server) Close() {
	for sfd := range s.sessions {
		unix.Close(sfd)
	}
}

// HandleEvent is an epoll.EventFn that handles events for echoing packets.
//
// The handler will determine if the event is for a client that has an existing
// session or not.
//
// A new socket and file descriptor is created for each new client session.
func (s *Server) HandleEvent(fd int) error {
	n, a, err := unix.Recvfrom(fd, s.buf, 0)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	return s.echo(fd, s.buf[:n], a)
}

func (s *Server) echo(fd int, data []byte, a unix.Sockaddr) error {
	if err := unix.Sendto(fd, data, 0, a); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
