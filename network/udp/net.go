package udp

import (
	"bufio"
	"io"
	"log"
	"net"
	"sync"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
)

// Network is a handel.Network implementation using UDP as its transport layer
type Network struct {
	sync.RWMutex
	udpSock   *net.UDPConn
	listeners []h.Listener
	quit      bool
	enc       network.Encoding
}

// NewNetwork creates Nework baked by udp protocol
func NewNetwork(addr string, enc network.Encoding) (*Network, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}

	udpSock, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	listeners := []h.Listener{}
	udpNet := &Network{sync.RWMutex{}, udpSock, listeners, false, enc}
	go udpNet.handler()
	return udpNet, nil
}

// Stop closes
func (udpNet *Network) Stop() {
	udpNet.Lock()
	defer udpNet.Unlock()
	udpNet.quit = true
}

//RegisterListener registers listener for processing incoming packets
func (udpNet *Network) RegisterListener(listener h.Listener) {
	udpNet.Lock()
	defer udpNet.Unlock()
	udpNet.listeners = append(udpNet.listeners, listener)
}

//Send sends a packet to supplied identities
func (udpNet *Network) Send(identities []h.Identity, packet *h.Packet) {
	for _, id := range identities {
		udpNet.send(id, packet)
	}
}

func (udpNet *Network) send(identity h.Identity, packet *h.Packet) {
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
	//fmt.Printf("%s -> sending packet to %s\n", udpSock.LocalAddr().String(), addr)
}

func (udpNet *Network) handler() {
	enc := udpNet.enc
	for {
		//udpNet.quit and udpNet.listeners have to be guarded by a read lock
		udpNet.RLock()
		quit := udpNet.quit
		udpNet.RUnlock()

		if quit {
			return
		}
		socket := udpNet.udpSock
		reader := bufio.NewReader(socket)
		var byteReader io.Reader = bufio.NewReader(reader)
		packet, err := enc.Decode(byteReader)
		if err != nil {
			log.Println(err)
			continue
		}
		udpNet.dispatch(packet)
	}
}

func (udpNet *Network) dispatch(p *handel.Packet) {
	udpNet.RLock()
	defer udpNet.RUnlock()
	for _, listener := range udpNet.listeners {
		//fmt.Printf("%s -> dispatching packet to listener %p!\n", udpNet.udpSock.LocalAddr().String(), listener)
		listener.NewPacket(p)
	}
}
