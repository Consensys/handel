package lib

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

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
	addr      string
	exp       int
	total     int
	n         *udp.Network
	readys    map[int]bool
	addresses map[string]bool
	done      bool
	waitAll   chan bool
}

// NewSyncMaster returns an SyncMaster that listens on the given address,
// for a expected number of READY messages.
func NewSyncMaster(addr string, expected, total int) *SyncMaster {
	n, err := udp.NewNetwork(addr, network.NewGOBEncoding())
	if err != nil {
		panic(err)
	}
	s := new(SyncMaster)
	n.RegisterListener(s)
	s.exp = expected
	s.n = n
	s.total = total
	s.addr = addr
	s.readys = make(map[int]bool)
	s.addresses = make(map[string]bool)
	s.waitAll = make(chan bool, 1)
	return s
}

// WaitAll returns a channel that is filled wen all the nodes have replied
// and the master have sent the START message.
func (s *SyncMaster) WaitAll() chan bool {
	return s.waitAll
}

// Reset the syncmaster to its initial step - new calls to WaitAll can be made.
func (s *SyncMaster) Reset() {
	s.Lock()
	defer s.Unlock()
	s.done = false
	s.readys = make(map[int]bool)
	s.addresses = make(map[string]bool)
	s.waitAll = make(chan bool, 1)
}

// NewPacket implements the Listener interface
func (s *SyncMaster) NewPacket(p *handel.Packet) {
	s.Lock()
	defer s.Unlock()
	if s.done {
		return
	}

	msg := new(syncMessage)
	if err := msg.FromBytes(p.MultiSig); err != nil {
		panic(err)
	}

	switch msg.State {
	case READY:
		s.handleReady(msg)
	default:
		panic("receiving something unexpected")
	}
}

func (s *SyncMaster) handleReady(incoming *syncMessage) {
	for _, id := range incoming.IDs {
		_, stored := s.readys[id]
		if !stored {
			s.readys[id] = true
		}
	}
	_, stored := s.addresses[incoming.Address]
	if !stored {
		s.addresses[incoming.Address] = true
	}
	fmt.Print(s.String())
	if len(s.readys) < s.exp {
		return
	}

	s.done = true
	// send the messagesssss
	msg := &syncMessage{State: START}
	buff, err := msg.ToBytes()
	if err != nil {
		panic(err)
	}
	packet := &handel.Packet{MultiSig: buff}
	ids := make([]handel.Identity, 0, len(s.addresses))
	for address := range s.addresses {
		id := handel.NewStaticIdentity(0, address, nil)
		ids = append(ids, id)
	}
	go func() {
		s.n.Send(ids, packet)
		s.waitAll <- true
	}()
}

func (s *SyncMaster) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Sync Master received %d/%d status\n", len(s.readys), s.exp)
	for id := 0; id < s.total; id++ {
		_, ok := s.readys[id]
		if !ok {
			fmt.Fprintf(&b, "\t- %d -absent-", id)
		} else {
			//for id, msg := range s.readys {
			//_, port, _ := net.SplitHostPort(msg.Address)
			fmt.Fprintf(&b, "\t- %d +finished+", id)
		}
		if (id+1)%4 == 0 {
			fmt.Fprintf(&b, "\n")
		}
	}
	fmt.Fprintf(&b, "\n")
	return b.String()
}

// Stop stops the network layer of the syncmaster
func (s *SyncMaster) Stop() {
	s.Lock()
	defer s.Unlock()
	s.n.Stop()
}

// SyncSlave sends its state to the master and waits for a START message
type SyncSlave struct {
	sync.Mutex
	own        string
	master     string
	net        *udp.Network
	ids        []int
	waitCh     chan bool
	done       bool
	internDone chan bool
	sendDone   chan bool
}

// NewSyncSlave returns a Sync to use as a node in the system to synchronize
// with the master
func NewSyncSlave(own, master string, ids []int) *SyncSlave {
	n, err := udp.NewNetwork(own, network.NewGOBEncoding())
	if err != nil {
		panic(err)
	}
	slave := new(SyncSlave)
	n.RegisterListener(slave)
	slave.ids = ids
	slave.net = n
	slave.own = own
	slave.master = master
	slave.waitCh = make(chan bool, 1)
	slave.internDone = make(chan bool, 1)
	slave.sendDone = make(chan bool, 1)
	go slave.sendReadyState()
	go slave.waitDone()
	return slave
}

const retrials = 5
const wait = 1 * time.Second

func (s *SyncSlave) sendReadyState() {
	for i := 0; i < retrials; i++ {
		msg := &syncMessage{State: READY, IDs: s.ids, Address: s.own}
		buff, err := msg.ToBytes()
		if err != nil {
			panic(err)
		}
		packet := &handel.Packet{MultiSig: buff}
		id := handel.NewStaticIdentity(0, s.master, nil)
		s.net.Send([]handel.Identity{id}, packet)
		time.Sleep(wait)
		select {
		case <-s.sendDone:
			return
		default:
			continue
		}
	}
}

// WaitMaster returns a channel that receives a value when the sync master sends
// the START message
func (s *SyncSlave) WaitMaster() chan bool {
	return s.waitCh
}

func (s *SyncSlave) waitDone() {
	<-s.internDone
	s.sendDone <- true
	s.waitCh <- true
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
	s.internDone <- true
}

// Reset re-initializes the syncslave to its initial state - it sends its status
// to the master and new calls to WaitMaster can be made.
func (s *SyncSlave) Reset() {
	s.Lock()
	defer s.Unlock()
	s.done = false
	go s.sendReadyState()
	go s.waitDone()

}

// Stop the network layer of the syncslave
func (s *SyncSlave) Stop() {
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
	State   int    // READY - START
	Address string // address of the slave
	IDs     []int  // ID of the slave - useful for debugging
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
