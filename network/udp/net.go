package udp

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
)

type Network interface {
	Send(identities  []h.Identity, packet *h.Packet)
	RegisterListener(listener h.Listener)
	getListeners() []handel.Listener
	Stop()
}

// UDPNetwork is a handel.Network implementation using UDP as its transport layer
// listens on 0.0.0.0
type UDPNetwork struct {
	sync.RWMutex
	udpSock   *net.UDPConn
	listeners []h.Listener
	quit      bool
	enc       network.Encoding
	newPacket chan *handel.Packet
	process   chan *handel.Packet
	ready     chan bool
	done      chan bool
	buff      []*handel.Packet
}

type delayedPacket struct {
	toSendAt	time.Time
	identities []h.Identity
	data		h.Packet
}

type DelayedUDPNetwork struct {
	network Network
	delay time.Duration
	in chan *delayedPacket
}

func (n *DelayedUDPNetwork) Send(identities  []h.Identity, packet *h.Packet) {
	n.in <- &delayedPacket{time.Now().Add(n.delay), identities,*packet}
}

func  (n *DelayedUDPNetwork) backgroundSend() {
	for dp := range n.in {
		delta := dp.toSendAt.Sub(time.Now())
		if delta.Nanoseconds() > time.Millisecond.Nanoseconds() {
			// Not really useful to sleep for less than 1ms
			time.Sleep(delta)
		}
		n.network.Send(dp.identities, &dp.data)
	}
}

func (n *DelayedUDPNetwork) Stop() {
	n.network.Stop()
	close(n.in)
}

func (n *DelayedUDPNetwork) RegisterListener(listener h.Listener) {
	n.network.RegisterListener(listener)
}

func (n *DelayedUDPNetwork) getListeners() []handel.Listener{
	return n.network.getListeners()
}

func NewDelayedUDPNetwork(delay time.Duration, addr string, enc network.Encoding) (*DelayedUDPNetwork, error) {
	n, err := NewNetwork(addr, enc)
	if err != nil {
		return nil, err
	}

	res :=  &DelayedUDPNetwork{
		n,
		delay,
		make(chan *delayedPacket, 10000),
	}

	go res.backgroundSend()

	return res, nil
}

// NewNetwork creates Network baked by udp protocol
func NewNetwork(addr string, enc network.Encoding) (Network, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	newAddr := net.JoinHostPort("0.0.0.0", port)

	// we have to bind to 0.0.0.0 (needed for AWS)
	udpAddr, err := net.ResolveUDPAddr("udp4", newAddr)
	if err != nil {
		return nil, err
	}

	udpSock, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	udpNet := &UDPNetwork{
		udpSock:   udpSock,
		enc:       enc,
		newPacket: make(chan *handel.Packet, 20000),
		process:   make(chan *handel.Packet, 100),
		ready:     make(chan bool, 1),
		done:      make(chan bool, 1),
	}

	go udpNet.handler()
	go udpNet.loop()
	go udpNet.dispatchLoop()
	return udpNet, nil
}

// Stop closes
func (udpNet *UDPNetwork) Stop() {
	udpNet.Lock()
	defer udpNet.Unlock()
	if udpNet.quit {
		return
	}
	udpNet.quit = true
	close(udpNet.done)
}

//RegisterListener registers listener for processing incoming packets
func (udpNet *UDPNetwork) RegisterListener(listener h.Listener) {
	udpNet.Lock()
	defer udpNet.Unlock()
	udpNet.listeners = append(udpNet.listeners, listener)
}

//Send sends a packet to supplied identities
func (udpNet *UDPNetwork) Send(identities []h.Identity, packet *h.Packet) {
	for _, id := range identities {
		udpNet.send(id, packet)
	}
}

func (udpNet *UDPNetwork) send(identity h.Identity, packet *h.Packet) {
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

	err = udpNet.enc.Encode(packet, byteWriter)
	if err != nil {
		//TODO consider changing it to error logging
		return
	}
	byteWriter.Flush()
	//fmt.Printf("%s -> sending packet to %s\n", udpSock.LocalAddr().String(), addr)
}

func (udpNet *UDPNetwork) handler() {
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
		//udpNet.dispatch(packet)
		udpNet.newPacket <- packet
	}
}

func (udpNet *UDPNetwork) loop() {
	pendings := list.New()
	var ready = false
	send := func() {
		if pendings.Len() == 0 {
			return
		}
		if !ready {
			return
		}
		toProcess := pendings.Remove(pendings.Front()).(*handel.Packet)
		udpNet.process <- toProcess
		ready = false
	}
	for {
		select {
		case newPacket := <-udpNet.newPacket:
			if len(newPacket.MultiSig) == 0 {
				fmt.Printf(" -- empty packet -- \n")
				continue
			}
			pendings.PushBack(newPacket)
			if ready {
				send()
			}
		case <-udpNet.ready:
			ready = true
			send()
		case <-udpNet.done:
			return
		}
	}
}

func (udpNet *UDPNetwork) getListeners() []handel.Listener {
	udpNet.RLock()
	defer udpNet.RUnlock()
	return udpNet.listeners
}

func (udpNet *UDPNetwork) dispatchLoop() {
	dispatch := func(p *handel.Packet) {
		listeners := udpNet.getListeners()
		for _, listener := range listeners {
			listener.NewPacket(p)
		}
	}

	udpNet.ready <- true
	for {
		select {
		case <-udpNet.done:
			return
		case newPacket := <-udpNet.process:
			// new packet to analyze
			dispatch(newPacket)
			// we say we're ready to analyze more
			udpNet.ready <- true
		}
	}
}
