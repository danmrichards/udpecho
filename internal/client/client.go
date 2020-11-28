package client

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/danmrichards/udpecho/internal/utils"
)

// EchoClient is a UDP client that sends packets to a given address and expects
// them to be echoed back.
type EchoClient struct {
	addr     *net.UDPAddr
	numConns int
	conns    []io.Closer
	wg       sync.WaitGroup
	done     chan struct{}
}

// NewEchoClient returns an echo client configured to make numConns connections
// to addr.
func NewEchoClient(addr string, numConns int) (*EchoClient, error) {
	ra, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	return &EchoClient{
		addr:     ra,
		numConns: numConns,
		done:     make(chan struct{}),
		conns:    make([]io.Closer, 0, numConns),
	}, nil
}

// Start starts echoing packets.
func (e *EchoClient) Start() error {
	for i := 0; i < e.numConns; i++ {
		c, err := net.DialUDP("udp", nil, e.addr)
		if err != nil {
			return fmt.Errorf("dial: %w", err)
		}
		e.conns = append(e.conns, c)

		// Send loop.
		e.wg.Add(1)
		go e.send(c)

		// Receive loop.
		e.wg.Add(1)
		go e.recv(c)
	}

	return nil
}

// Stop stops the echo client.
func (e *EchoClient) Stop() error {
	close(e.done)

	for _, c := range e.conns {
		if err := c.Close(); err != nil {
			log.Println("close conn:", err)
		}
	}

	e.wg.Wait()

	return nil
}

// send writes a random message to c until Stop is called.
func (e *EchoClient) send(c net.Conn) {
	defer e.wg.Done()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for !utils.IsDone(e.done) {
		msg := fmt.Sprintf("foo %d", r.Int())
		if _, err := c.Write([]byte(msg)); err != nil {
			if utils.IsDone(e.done) {
				// Ignore the error if we've closed the connection.
				return
			}

			log.Println("write:", err)
			continue
		}

		log.Printf("send: from: %q data: %q\n", c.LocalAddr(), msg)
		time.Sleep(100 * time.Millisecond)
	}
}

// recv receives messages from c until Stop is called.
func (e *EchoClient) recv(c net.Conn) {
	defer e.wg.Done()

	buf := make([]byte, 1024)
	for !utils.IsDone(e.done) {
		n, err := c.Read(buf)
		if err != nil {
			if utils.IsDone(e.done) {
				// Ignore the error if we've closed the connection.
				return
			}

			log.Println(err)
			continue
		}

		log.Printf("recv: to: %q data: %q\n", c.LocalAddr(), string(buf[:n]))
	}
}
