package udp

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"

	h "github.com/ConsenSys/handel"
)

type udpNet struct {
	listeners *[]h.Listener
}

// NewUDPNetwork creates Nework baked by udp protocol
func NewUDPNetwork(listenPort int) *udpNet {
	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	listeners := &[]h.Listener{}
	go handler(listeners, conn)
	return &udpNet{listeners}
}

//RegisterListener registers listener for processing incoming packets
func (udpNet *udpNet) RegisterListener(listener h.Listener) {
	*udpNet.listeners = append(*udpNet.listeners, listener)
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

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	byteWriter := bufio.NewWriter(conn)
	// The packets are "gob" encoded
	enc := gob.NewEncoder(byteWriter)
	err = enc.Encode(packet)
	if err != nil {
		//TODO consider changing it to error logging
		panic(err)
	}
	byteWriter.Flush()
}

func handler(listeners *[]h.Listener, conn *net.UDPConn) {
	for {
		packetHandler(listeners, conn)
	}
}

func packetHandler(listeners *[]h.Listener, conn *net.UDPConn) {
	reader := bufio.NewReader(conn)
	var byteReader io.Reader = bufio.NewReader(reader)
	var packet h.Packet
	//Decode gob encoded packet
	dec := gob.NewDecoder(byteReader)
	err := dec.Decode(&packet)
	if err != nil {
		log.Println(err)
	}
	for _, listener := range *listeners {
		listener.NewPacket(&packet)
	}
}
