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
		go func(i int) {
			send(c, i)
		}(i)

		// Receive loop.
		go func(i int) {
			recv(c, i)
		}(i)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c
}

func send(c net.Conn, i int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		msg := fmt.Sprintf("conn %d %d", i, r.Int())
		if _, err := c.Write([]byte(msg)); err != nil {
			log.Println("write:", err)
			continue
		}

		log.Printf("conn %d send: from: %q data: %q\n", i, c.LocalAddr(), msg)
		time.Sleep(100 * time.Millisecond)
	}
}

func recv(c net.Conn, i int) {
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		data := string(buf[:n])

		var ci, r int
		fmt.Sscanf(data, "conn %d %d", &ci, &r)

		if ci != i {
			log.Fatalf("packet mismatch: received for conn %d on conn %d", ci, i)
		}

		log.Printf("conn %d recv: to: %q data: %q\n", i, c.LocalAddr(), data)
	}
}
