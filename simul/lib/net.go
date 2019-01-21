package lib

import (
	"net"
	"strconv"
	"time"
)

// GetFreePort returns a free tcp port or panics
func GetFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer func() {
		l.Close()
		time.Sleep(2 * time.Millisecond)
	}()
	return l.Addr().(*net.TCPAddr).Port
}

// GetFreeUDPPort returns a free usable UDP address
func GetFreeUDPPort() int {
	for i := 0; i < 1000; i++ {
		udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
		if err != nil {
			continue
		}
		sock, err := net.ListenUDP("udp4", udpAddr)
		if err != nil {
			continue
		}
		addr := sock.LocalAddr().String()
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		portI, err := strconv.Atoi(port)
		if err != nil {
			continue
		}
		defer func() { sock.Close(); time.Sleep(2 * time.Millisecond) }()
		return portI
	}
	panic("not found")
}
