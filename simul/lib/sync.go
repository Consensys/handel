package lib

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/ConsenSys/handel/network/udp"
)

// SyncMaster is a struct that handles the synchronization of all launched binaries
// by first expecting a message from each one of them, then sending them back a
// "START" message when all are ready. It uses UDP.
// The "Protocol" looks like this:
// - the SyncMaster listens on a UDP socket
// - each node sends a "READY" message to the starter over that socket.
// - the SyncMaster waits for n different READY messages.
// - once that is done, the SyncMaster sends a START message to all nodes.
//
// A READY message is a Packet which contains a structure inside the MultiSig
// field, as to re-use the UDP code already present.
type SyncMaster struct {
	sync.Mutex
	addr     string
	exp      int
	n        *udp.Network
	readys   map[string]bool
	done     bool
	waitDone chan bool
}

// NewSyncMaster returns an SyncMaster that listens on the given address,
// for a expected number of READY messages.
func NewSyncMaster(addr string, expected int) *SyncMaster {
	n, err := udp.NewNetwork(addr, network.NewGOBEncoding())
	if err != nil {
		panic(err)
	}
	s := new(SyncMaster)
	n.RegisterListener(s)
	s.exp = expected
	s.n = n
	s.addr = addr
	s.readys = make(map[string]bool)
	s.waitDone = make(chan bool, 1)
	return s
}

// WaitAllSetup returns a channel that is filled wen all the nodes have replied
// and the master have sent the START message.
func (s *SyncMaster) WaitAllSetup() chan bool {
	return s.waitDone
}

// NewPacket implements the Listener interface
func (s *SyncMaster) NewPacket(p *handel.Packet) {
	s.Lock()
	defer s.Unlock()
	defer s.checkStart()
	msg := new(syncMessage)
	if err := msg.FromBytes(p.MultiSig); err != nil {
		panic(err)
	}
	if msg.State != READY {
		panic("receiving something unexpected")
	}
	_, stored := s.readys[msg.Addr]
	if !stored {
		s.readys[msg.Addr] = true
	}
}

// checkStart looks if everybody have sent the READY message
func (s *SyncMaster) checkStart() {
	if s.done {
		return
	}
	if len(s.readys) < s.exp {
		return
	}
	s.done = true
	// send the messagesssss
	msg := &syncMessage{State: START, Addr: s.addr}
	buff, err := msg.ToBytes()
	if err != nil {
		panic(err)
	}
	packet := &handel.Packet{MultiSig: buff}
	ids := make([]handel.Identity, 0, s.exp)
	for addr := range s.readys {
		id := handel.NewStaticIdentity(0, addr, nil)
		ids = append(ids, id)
	}
	go func() {
		s.n.Send(ids, packet)
		s.n.Stop()
		s.waitDone <- true
	}()
}

// SyncSlave sends its state to the master and waits for a START message
type SyncSlave struct {
	sync.Mutex
	own    string
	master string
	net    *udp.Network
	waitCh chan bool
	done   bool
}

// NewSyncSlave returns a Sync to use as a node in the system to synchronize
// with the master
func NewSyncSlave(own, master string) *SyncSlave {
	n, err := udp.NewNetwork(own, network.NewGOBEncoding())
	if err != nil {
		panic(err)
	}
	slave := new(SyncSlave)
	n.RegisterListener(slave)
	slave.net = n
	slave.own = own
	slave.master = master
	slave.waitCh = make(chan bool, 1)
	go slave.sendReadyState()
	return slave
}

func (s *SyncSlave) sendReadyState() {
	msg := &syncMessage{State: READY, Addr: s.own}
	buff, err := msg.ToBytes()
	if err != nil {
		panic(err)
	}
	packet := &handel.Packet{MultiSig: buff}
	id := handel.NewStaticIdentity(0, s.master, nil)
	go s.net.Send([]handel.Identity{id}, packet)
	fmt.Printf("%s -> sending ready state to %s\n", s.own, s.master)
}

// WaitStart returns a channel that receives a value when the sync master sends
// the START message
func (s *SyncSlave) WaitStart() chan bool {
	return s.waitCh
}

// NewPacket implements the Listener interface
func (s *SyncSlave) NewPacket(p *handel.Packet) {
	s.Lock()
	defer s.Unlock()
	if s.done == true {
		return
	}

	msg := new(syncMessage)
	if err := msg.FromBytes(p.MultiSig); err != nil {
		panic(err)
	}

	if msg.State != START {
		panic("that should not happen")
	}
	s.done = true
	s.waitCh <- true

	s.net.Stop()
}

const (
	// READY state
	READY = iota
	// START state
	START
)

// syncMessage is what is sent between a SyncMaster and a SyncSlave
type syncMessage struct {
	State int    // READY - START
	Addr  string // address of the sender
}

func (s *syncMessage) ToBytes() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(s)
	return b.Bytes(), err
}

func (s *syncMessage) FromBytes(buff []byte) error {
	var b = bytes.NewBuffer(buff)
	dec := gob.NewDecoder(b)
	return dec.Decode(s)
}

// FindFreeUDPAddress returns a free usable UDP address
func FindFreeUDPAddress() string {
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
		sock.Close()
		return addr
	}
	panic("not found")
}
