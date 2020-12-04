package epoll

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
)

const (
	MaxEpollEvents = 1024
)

type EventFn func(fd int) error

type Poller struct {
	fd     int
	events []syscall.EpollEvent
	ef     EventFn
}

func NewPoller() (*Poller, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &Poller{
		fd:     fd,
		events: make([]syscall.EpollEvent, MaxEpollEvents),
	}, nil
}

func (p *Poller) HandlePacketConn(conn net.PacketConn, ef EventFn) (err error) {
	p.ef = ef

	var fd int
	fd, err = connFD(conn)
	if err != nil {
		return err
	}

	return p.Add(fd)
}

func (p *Poller) Close() error {
	return syscall.Close(p.fd)
}

func (p *Poller) Wait() error {
	for {
		n, err := syscall.EpollWait(p.fd, p.events, -1)
		if err != nil {
			var serr syscall.Errno
			if errors.As(err, &serr) && serr.Temporary() {
				err = nil
			} else {
				return err
			}
		}

		for i := 0; i < n; i++ {
			evt := p.events[i]
			if int(evt.Fd) == p.fd {
				// Connection closed.
				return nil
			}

			if evt.Events&syscall.EPOLLIN == 0 {
				continue
			}

			// TODO(dr): Other events? Disconnect?

			if err := p.ef(int(evt.Fd)); err != nil {
				log.Println(err)
			}
		}
	}
}

func (p *Poller) Add(fd int) error {
	return syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
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
