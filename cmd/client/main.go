package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/danmrichards/udpecho/internal/client"
)

var (
	host, port string
	conns      int
)

func main() {
	flag.StringVar(&host, "host", "127.0.0.1", "host of the relay server")
	flag.StringVar(&port, "port", "8888", "port of the relay server")
	flag.IntVar(&conns, "conns", 10, "number of connections to create")
	flag.Parse()

	ec, err := client.NewEchoClient(net.JoinHostPort(host, port), conns)
	if err != nil {
		log.Fatal(err)
	}

	if err = ec.Start(); err != nil {
		log.Println(err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("shutting down")
	if err = ec.Stop(); err != nil {
		log.Println(err)
	}
}
