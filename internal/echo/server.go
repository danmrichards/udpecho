package echo

import (
	"fmt"
	"net"
	"syscall"

	"github.com/danmrichards/udpecho/internal/epoll"
	"github.com/danmrichards/udpecho/internal/sock"
)

type Server struct {
	sessions map[int]string
	conn     net.PacketConn
	connsock syscall.Sockaddr
	poller   *epoll.Poller
	buf      []byte
}

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

func (s *Server) HandleEvent(fd int) error {
	// Check if we have a session for this client.
	if _, ok := s.sessions[fd]; !ok {
		n, a, err := s.conn.ReadFrom(s.buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		// Client socket address.
		csa, err := sock.Addr(a.String())
		if err != nil {
			return fmt.Errorf("client sock address: %w", err)
		}

		// Client socket.
		cfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
		if err != nil {
			return fmt.Errorf("new udp client socket: %w", err)
		}
		if err = syscall.SetsockoptInt(cfd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			return fmt.Errorf("client socket reuse: %w", err)
		}

		if err := syscall.Bind(cfd, s.connsock); err != nil {
			return fmt.Errorf("client socket bind: %w", err)
		}

		if err := syscall.Connect(cfd, csa); err != nil {
			return fmt.Errorf("client socket connect: %w", err)
		}

		// Start epoll watching new socket.
		if err = s.poller.Add(cfd); err != nil {
			return err
		}

		s.sessions[cfd] = a.String()

		return s.echo(cfd, s.buf[:n])
	}

	return s.echo(fd, nil)
}

func (s *Server) echo(cfd int, data []byte) error {
	if data == nil {
		n, err := syscall.Read(cfd, s.buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		data = s.buf[:n]
	}

	if _, err := syscall.Write(cfd, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
