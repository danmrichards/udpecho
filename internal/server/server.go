package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/mailru/easygo/netpoll"
)

// EchoServer is a UDP server that echos packets back to the sender.
type EchoServer struct {
	c    *net.UDPConn
	p    netpoll.Poller
	d    *netpoll.Desc
	done chan struct{}
	ps   *http.Server
	wg   sync.WaitGroup
}

// Option is a functional option that modifies the echo server.
type Option func(*EchoServer)

// WithProfile enables HTTP profiling.
func WithProfiling(addr string) Option {
	return func(es *EchoServer) {
		es.ps = &http.Server{
			Addr: addr,
		}
	}
}

// NewEchoServer returns a new echo server configured with the given options.
func NewEchoServer(addr string, opts ...Option) (es *EchoServer, err error) {
	// Configure to create as many workers as we have available threads. Leaving
	// 1 free for the main thread. Attempt to parallelise as much as we can.
	es = &EchoServer{
		done: make(chan struct{}),
	}

	for _, o := range opts {
		o(es)
	}

	es.p, err = netpoll.New(nil)
	if err != nil {
		return nil, fmt.Errorf("netpoll: %w", err)
	}

	ua, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve udp address: %w", err)
	}

	es.c, err = net.ListenUDP("udp", ua)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	es.d, err = netpoll.HandleRead(es.c)
	if err != nil {
		return nil, fmt.Errorf("handle read: %w", err)
	}

	return es, nil
}

// Server starts the echo server.
func (e *EchoServer) Serve() error {
	// Start the profiling server if required.
	if e.ps != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			log.Println("profiling:", e.ps.ListenAndServe())
		}()
	}

	buf := make([]byte, 1024)
	return e.p.Start(e.d, e.recv(buf))
}

// Stop stops the echo server.
func (e *EchoServer) Stop() error {
	close(e.done)
	if err := e.c.Close(); err != nil {
		return err
	}

	// Stop the profiling server if required.
	if e.ps != nil {
		if err := e.ps.Close(); err != nil {
			return err
		}
	}

	e.wg.Wait()

	return nil
}

func (e *EchoServer) recv(buf []byte) netpoll.CallbackFn {
	return func(evt netpoll.Event) {
		if evt&netpoll.EventReadHup != 0 {
			e.p.Stop(e.d)
			return
		}

		dd, err := netpoll.HandleRead(e.c)
		if err != nil {
			log.Println(err)
			return
		}

		if err = e.p.Start(dd, func(ev netpoll.Event) {
			if ev&netpoll.EventReadHup != 0 {
				e.p.Stop(dd)
				return
			}

			n, s, err := e.c.ReadFromUDP(buf)
			if err != nil {
				log.Println(err)
				return
			}

			if _, err = e.c.WriteToUDP(buf[:n], s); err != nil {
				log.Println(err)
			}
		}); err != nil {
			log.Println(err)
		}
	}
}
