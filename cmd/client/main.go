package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"time"
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

	ra, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, port))
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < conns; i++ {
		c, err := net.DialUDP("udp", nil, ra)
		if err != nil {
			log.Fatal(err)
		}

		// Send loop.
		go send(c)

		// Receive loop.
		go recv(c)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c
}

func send(c net.Conn) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		msg := fmt.Sprintf("foo %d", r.Int())
		if _, err := c.Write([]byte(msg)); err != nil {
			log.Println("write:", err)
			continue
		}

		log.Printf("send: from: %q data: %q\n", c.LocalAddr(), msg)
		time.Sleep(100 * time.Millisecond)
	}
}

func recv(c net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("recv: to: %q data: %q\n", c.LocalAddr(), string(buf[:n]))
	}
}
