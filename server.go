package main

import (
	"context"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"syscall"

	"github.com/danmrichards/udpecho/internal/echo"
	"github.com/danmrichards/udpecho/internal/epoll"
)

var addr = ":8888"

func main() {
	go func() { log.Println(http.ListenAndServe(":6060", nil)) }()

	var err error
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
		},
	}
	conn, err := lc.ListenPacket(context.Background(), "udp4", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", addr)

	p, err := epoll.NewPoller()
	if err != nil {
		log.Fatal(err)
	}

	s, err := echo.NewServer(conn, p)
	if err != nil {
		log.Fatal(err)
	}

	if err = p.HandlePacketConn(conn, s.HandleEvent); err != nil {
		log.Fatal(err)
	}

	log.Println(p.Wait())
}
