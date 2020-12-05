package epoll

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

const (
	MaxEpollEvents = 1024
)

// EventFn is a function that reacts to a epoll event.
type EventFn func(fd int) error

// Poller leverages epoll to handle packet connections.
type Poller struct {
	fd     int
	evFD   int
	events []unix.EpollEvent
	ef     EventFn
}

// NewPoller returns a new poller.
func NewPoller() (*Poller, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	p := &Poller{
		fd:     fd,
		events: make([]unix.EpollEvent, MaxEpollEvents),
	}

	// Set finalizer for write end of socket pair to avoid data races when
	// closing Epoll instance and EBADF errors on writing ctl bytes from callers.
	r0, _, errno := unix.Syscall(unix.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	p.evFD = int(r0)
	if err = p.Add(p.evFD); err != nil {
		unix.Close(fd)
		unix.Close(p.evFD)
		return nil, err
	}

	return p, nil
}

// HandlePacketConn configures the poller to dispatch the given event function
// for events triggered by the packet connection.
func (p *Poller) HandlePacketConn(conn net.PacketConn, ef EventFn) (err error) {
	p.ef = ef

	var fd int
	fd, err = connFD(conn)
	if err != nil {
		return err
	}

	if err = p.Add(fd); err != nil {
		unix.Close(fd)
		return err
	}

	return nil
}

// Close closes the poller.
func (p *Poller) Close() error {
	return unix.Close(p.fd)
}

// Wait waits for events to be triggered by epoll.
func (p *Poller) Wait() error {
	for {
		n, err := unix.EpollWait(p.fd, p.events, -1)
		if err != nil {
			var serr unix.Errno
			if errors.As(err, &serr) && serr.Temporary() {
				err = nil
			} else {
				return err
			}
		}

		for i := 0; i < n; i++ {
			evt := p.events[i]
			if int(evt.Fd) == p.evFD {
				// Connection closed.
				return nil
			}

			// TODO(dr): Other events? Disconnect?
			switch {
			case evt.Events&unix.EPOLLERR != 0:
				return fmt.Errorf("error: %+v", evt)
			case evt.Events&unix.EPOLLOUT != 0:
			case evt.Events&unix.EPOLLIN != 0:
				if p.ef == nil {
					continue
				}
				if err = p.ef(int(evt.Fd)); err != nil {
					log.Println("ef error:", err)
				}
			}
		}
	}
}

// Add adds a new file descriptor to the list watched by epoll.
func (p *Poller) Add(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLRDHUP,
		Fd:     int32(fd),
	})
}

func connFD(c interface{}) (int, error) {
	f, ok := c.(interface {
		File() (*os.File, error)
	})
	if !ok {
		return 0, fmt.Errorf("does not implement File method")
	}

	cf, err := f.File()
	if err != nil {
		return 0, err
	}

	return int(cf.Fd()), nil
}
