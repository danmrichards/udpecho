package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"

	"github.com/danmrichards/udpecho/internal/utils"
)

// EchoServer is a UDP server that echos packets back to the sender.
type EchoServer struct {
	c       net.PacketConn
	workers int
	done    chan struct{}
	wg      sync.WaitGroup
	ps      *http.Server
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
		workers: runtime.GOMAXPROCS(0) - 1,
		done:    make(chan struct{}),
	}

	for _, o := range opts {
		o(es)
	}

	// Free up a thread for profiling if enabled.
	if es.ps != nil {
		es.workers--
	}

	es.c, err = net.ListenPacket("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	return es, nil
}

// Server starts the echo server.
func (e *EchoServer) Serve() {
	// Start the profiling server if required.
	if e.ps != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			log.Println("profiling:", e.ps.ListenAndServe())
		}()
	}

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.recv()
	}
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

func (e *EchoServer) recv() {
	defer e.wg.Done()

	buf := make([]byte, 1024)
	for !utils.IsDone(e.done) {
		n, s, err := e.c.ReadFrom(buf)
		if err != nil {
			if utils.IsDone(e.done) {
				// Ignore the error if we've closed the connection.
				return
			}

			log.Println("read UDP:", err)
			return
		}

		if _, err = e.c.WriteTo(buf[:n], s); err != nil {
			log.Println("write UDP:", err)
		}
	}
}
