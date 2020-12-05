package sock

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// Addr returns a socket address for the given UDP address.
func Addr(addr string) (unix.Sockaddr, error) {
	udp, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP address: %w", err)
	}

	sa := &unix.SockaddrInet4{Port: udp.Port}
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
func ConnectClient(sa unix.Sockaddr, client string) (int, error) {
	// Create new socket, setting the reuse address flag as it will be on the
	// same address as the server socket.
	cfd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return -1, fmt.Errorf("new udp client socket: %w", err)
	}

	if err = unix.SetsockoptInt(cfd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return -1, fmt.Errorf("client socket reuse: %w", err)
	}

	// As soon as we Bind the new socket below, an opportunity for a race
	// condition arises. Under heavy load it is possible that datagrams can be
	// sent via this new socket before we have had a chance to Connect it to the
	// client address. The Connect will cause incoming packets to be filtered to
	// just the specified client, and also set the default write address. This
	// race results in packets coming in from other clients to this socket and
	// the application cannot detect this as being the case as we will have made
	// the Connect call by this point, making it seem like the socket is correct.
	//
	// Close this race by leveraging the Berkeley Packet Filtering (BPF) virtual
	// machine. We set up a filter to drop all packets on the new socket as soon
	// as it's created. Only once the Bind and Connect are complete do we drop the
	// filter.
	//
	// See: https://www.kernel.org/doc/Documentation/networking/filter.txt
	fprog := &unix.SockFprog{
		Len: 1,
		Filter: &unix.SockFilter{
			Code: 0x06,
			Jt:   0,
			Jf:   0,
			K:    0x00000000,
		},
	}
	if err = unix.SetsockoptSockFprog(cfd, unix.SOL_SOCKET, unix.SO_ATTACH_FILTER, fprog); err != nil {
		return -1, fmt.Errorf("client socket attach filter: %w", err)
	}

	// Bind to the server and connect the client.
	if err = unix.Bind(cfd, sa); err != nil {
		return -1, fmt.Errorf("client socket bind: %w", err)
	}

	// Client socket address.
	csa, err := Addr(client)
	if err != nil {
		return -1, fmt.Errorf("client sock address: %w", err)
	}

	if err = unix.Connect(cfd, csa); err != nil {
		return -1, fmt.Errorf("client socket connect: %w", err)
	}

	// Socket is now bound to the server and filtered by just our client address.
	// It is now safe to remove the filter.
	if err = unix.SetsockoptInt(cfd, unix.SOL_SOCKET, unix.SO_DETACH_FILTER, 0); err != nil {
		return -1, fmt.Errorf("client socket detach filter: %w", err)
	}

	return cfd, nil
}
