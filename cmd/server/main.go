package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/danmrichards/udpecho/internal/server"
)

var (
	port, profilePort string
)

func main() {
	flag.StringVar(&port, "port", "8888", "port to bind the server on")
	flag.StringVar(&profilePort, "profile", "", "port on which to bind the profile server (disabled if blank)")
	flag.Parse()

	opts := make([]server.Option, 0, 1)
	if profilePort != "" {
		opts = append(opts, server.WithProfiling(net.JoinHostPort(
			"", profilePort,
		)))
	}

	addr := net.JoinHostPort("", port)
	es, err := server.NewEchoServer(addr, opts...)
	if err != nil {
		log.Fatal("listen UDP", err)
	}
	fmt.Println("listening on", addr)

	es.Serve()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("shutting down")
	if err = es.Stop(); err != nil {
		log.Println(err)
	}
}
