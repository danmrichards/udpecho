# UDP Echo
A simple server/client pair that echoes UDP packets to each other.

## Installation
```bash
$ go get -u github.com/danmrichards/udpecho/cmd/...
```

## Usage
Server
```
Usage of ./bin/server-linux-amd64:
  -port string
        port to bind the server on (default "8888")
  -profile string
        port on which to bind the profile server (disabled if blank)
```

Client
```
Usage of ./bin/client-linux-amd64:
  -conns int
        number of connections to create (default 10)
  -host string
        host of the relay server (default "127.0.0.1")
  -port string
        port of the relay server (default "8888")
```

## Building From Source
Clone this repo and build the binaries:
```bash
$ make build
```
