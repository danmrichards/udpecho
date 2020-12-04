package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"syscall"

	"github.com/danmrichards/udpecho/internal/echo"
	"github.com/danmrichards/udpecho/internal/epoll"
)

var port, profilePort string

func main() {
	flag.StringVar(&port, "port", "8888", "port to bind the server on")
	flag.StringVar(&profilePort, "profile", "", "port on which to bind the profile server (disabled if blank)")
	flag.Parse()

	if profilePort != "" {
		go func() {
			log.Println(
				http.ListenAndServe(net.JoinHostPort("", profilePort), nil),
			)
		}()
	}

	addr := net.JoinHostPort("", port)

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
