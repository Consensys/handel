package udp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
)

type udpNet struct {
	sync.RWMutex
	udpSock   *net.UDPConn
	listeners []h.Listener
	quit      bool
	enc       network.Encoding
}

// NewUDPNetwork creates Nework baked by udp protocol
func NewUDPNetwork(listenPort int, enc network.Encoding) *udpNet {
	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(err)
	}

	udpSock, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	listeners := []h.Listener{}
	udpNet := &udpNet{sync.RWMutex{}, udpSock, listeners, false, enc}
	go udpNet.handler()
	return udpNet
}

func (udpNet *udpNet) Stop() {
	udpNet.Lock()
	defer udpNet.Unlock()
	udpNet.quit = true
}

//RegisterListener registers listener for processing incoming packets
func (udpNet *udpNet) RegisterListener(listener h.Listener) {
	udpNet.Lock()
	defer udpNet.Unlock()
	udpNet.listeners = append(udpNet.listeners, listener)
}

//Send sends a packet to supplied identities
func (udpNet *udpNet) Send(identities []h.Identity, packet *h.Packet) {
	for _, id := range identities {
		udpNet.send(id, packet)
	}
}

func (udpNet *udpNet) send(identity h.Identity, packet *h.Packet) {
	addr := identity.Address()
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		//TODO consider changing it to error logging
		panic(err)
	}

	udpSock, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		panic(err)
	}
	defer udpSock.Close()

	byteWriter := bufio.NewWriter(udpSock)
	// The packets are "gob" encoded
	//	enc := gob.NewEncoder(byteWriter)
	//	err = enc.Encode(packet)

	udpNet.enc.Encode(packet, byteWriter)
	if err != nil {
		//TODO consider changing it to error logging
		panic(err)
	}
	byteWriter.Flush()
}

func (udpNet *udpNet) handler() {
	for {
		//udpNet.quit and udpNet.listeners have to be guarded by a read lock
		udpNet.RLock()
		quit := udpNet.quit
		listeners := udpNet.listeners
		enc := udpNet.enc
		udpNet.RUnlock()

		if quit {
			return
		}
		packetHandler(listeners, udpNet.udpSock, enc)
	}
}

func packetHandler(listeners []h.Listener, udpSock *net.UDPConn, enc network.Encoding) {
	reader := bufio.NewReader(udpSock)
	var byteReader io.Reader = bufio.NewReader(reader)

	packet, err := enc.Decode(byteReader)

	if err != nil {
		log.Println(err)
	}
	for _, listener := range listeners {
		listener.NewPacket(packet)
	}
}
