package sock

import (
	"fmt"
	"net"
	"syscall"
)

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
