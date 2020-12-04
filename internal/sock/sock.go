package sock

import (
	"fmt"
	"net"
	"syscall"
)

// Addr returns a socket address for the given UDP address.
func Addr(addr string) (syscall.Sockaddr, error) {
	udp, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP address: %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: udp.Port}
	if udp.IP != nil {
		if len(udp.IP) == 16 {
			copy(sa.Addr[:], udp.IP[12:16]) // copy last 4 bytes of slice to array
		} else {
			copy(sa.Addr[:], udp.IP) // copy all bytes of slice to array
		}
	}

	return sa, nil
}

// ConnectClient connects the given client address to the server socket and
// returns the file descriptor of the new client socket.
func ConnectClient(sa syscall.Sockaddr, client string) (int, error) {
	// Client socket address.
	csa, err := Addr(client)
	if err != nil {
		return -1, fmt.Errorf("client sock address: %w", err)
	}

	// Create new socket, setting the reuse address flag as it will be on the
	// same address as the server socket.
	cfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return -1, fmt.Errorf("new udp client socket: %w", err)
	}
	if err = syscall.SetsockoptInt(cfd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return -1, fmt.Errorf("client socket reuse: %w", err)
	}

	// Bind to the server and connect the client.
	if err := syscall.Bind(cfd, sa); err != nil {
		return -1, fmt.Errorf("client socket bind: %w", err)
	}
	if err := syscall.Connect(cfd, csa); err != nil {
		return -1, fmt.Errorf("client socket connect: %w", err)
	}

	return cfd, nil
}