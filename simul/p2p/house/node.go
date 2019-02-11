package house

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/ConsenSys/handel/network/udp"
	"github.com/ConsenSys/handel/simul/lib"
)

const handelTopic = "handel"

// Node implements gossiping
type Node struct {
	sync.Mutex
	gossipsOut map[string]chan Gossip
	counter    *network.CounterEncoding
	n          *lib.Node
	net        handel.Network
	reg        handel.Registry
	own        *Gossip
	fanout     int
	period     time.Duration
	topics     map[string]*topic
}

// Gossip is the packet we send and receive
type Gossip struct {
	Topic string
	ID    int32
	Msg   []byte
}

// gossipState holds the information about a specific message (gossip) under a
// topic - how many times have we retransmitted it, etc
type gossipState struct {
	sync.Mutex
	// message to transmit
	gossip *Gossip
	// bitset of peers we have transmitted the message
	sentTo        handel.BitSet
	retransmitted int
	fanout        int
	// id of the node holding this state
	ourID int32
	// id of the sender of the gossip message
	senderID int32
}

// Update returns a list of IDs to send the returned gossip as well.
func (g *gossipState) Update() (*Gossip, []int) {
	// find the ones not sent to yet
	ids := make([]int, 0, g.fanout)
	size := g.sentTo.BitLength()
	for len(ids) < g.fanout {
		i := rand.Intn(size)
		if g.sentTo.Get(i) {
			continue
		}
		if int32(i) == g.ourID {
			continue
		}
		g.sentTo.Set(i, true)
		ids = append(ids, i)
	}
	return g.gossip, ids
}

type topic struct {
	sync.Mutex
	send    func([]int32, *Gossip)
	gossip  *Gossip
	ourID   int32
	period  time.Duration
	fanout  int
	size    int
	topic   string
	seen    map[int32]handel.BitSet
	gossips map[int32]*Gossip
	done    chan bool
	isDone  bool
}

func newTopic(ourID int32, topicID string, size, fanout int, period time.Duration,
	fn func([]int32, *Gossip)) *topic {
	s := &topic{
		ourID:   ourID,
		topic:   topicID,
		size:    size,
		seen:    make(map[int32]handel.BitSet),
		gossips: make(map[int32]*Gossip),
		done:    make(chan bool),
		period:  period,
		send:    fn,
		fanout:  fanout,
	}
	go s.loop()
	return s
}

func (s *topic) loop() {
	t := time.NewTicker(s.period)
	for {
		select {
		case <-t.C:
			s.SendUpdates()
		case <-s.done:
			return
		}
	}
}

func (s *topic) stop() {
	s.Lock()
	defer s.Unlock()
	if s.isDone {
		return
	}
	close(s.done)
	s.isDone = true
}

// save the gossip
func (s *topic) NewGossip(g *Gossip) bool {
	s.Lock()
	defer fmt.Println(s.ourID, "quit NewGossip")
	defer s.Unlock()
	if s.isDone {
		return false
	}
	bs := s.getOrCreate(g.ID)
	fmt.Println(s.ourID, "-", s.topic, " - received NEW gossip of ", g.ID, "(got it?", bs.Get(int(g.ID)))
	if bs.Get(int(g.ID)) {
		return false
	}
	// we must re-broadcast if we did not see it before
	// epidemic !
	bs.Set(int(g.ID), true)
	s.gossips[g.ID] = g
	ids := s.chooseIDs(g.ID)
	s.send(ids, g)
	fmt.Println(s.ourID, "-", s.topic, " - rebroadcasted gossip of ", g.ID, " to ", ids)
	return true
}

func (s *topic) getOrCreate(gID int32) handel.BitSet {
	bs, exists := s.seen[gID]
	if !exists {
		bs = handel.NewWilffBitset(s.size)
		s.seen[gID] = bs
	}
	return bs
}

// sendUpdate sends update about our own gossip and the ones we received
func (s *topic) SendUpdates() {
	s.Lock()
	defer fmt.Println(s.ourID, "quit sent updates")
	defer s.Unlock()
	for id, g := range s.gossips {
		ids := s.chooseIDs(id)
		fmt.Println(s.topic, s.ourID, " - send updates of ", id, "to ", ids)
		s.send(ids, g)
	}
}

// chooseIDs returns new ids not rebroadcasted yet for this id
func (s *topic) chooseIDs(gossipID int32) []int32 {
	bs := s.getOrCreate(gossipID)
	// find the ones not sent to yet
	ids := make([]int32, 0, s.fanout)
	for len(ids) < s.fanout {
		i := rand.Intn(s.size)
		if bs.Get(i) {
			continue
		}
		if int32(i) == s.ourID {
			continue
		}
		bs.Set(i, true)
		ids = append(ids, int32(i))
	}
	return ids
}

// NewNode returns
func NewNode(n *lib.Node, reg handel.Registry, fanout int, period time.Duration) (*Node, error) {
	addr := n.Identity.Address()
	enc := network.NewCounterEncoding(network.NewGOBEncoding())
	net, err := udp.NewNetwork(addr, enc)
	if err != nil {
		return nil, err
	}

	node := &Node{
		n:          n,
		counter:    enc,
		net:        net,
		reg:        reg,
		period:     period,
		fanout:     fanout,
		topics:     make(map[string]*topic),
		gossipsOut: make(map[string]chan Gossip),
	}
	net.RegisterListener(node)
	return node, nil
}

// Identity id
func (n *Node) Identity() handel.Identity {
	return n.n.Identity
}

// SecretKey secret
func (n *Node) SecretKey() lib.SecretKey {
	return n.n.SecretKey
}

var setupMsg = []byte{0x01, 0x02, 0x03, 0x04}

// Diffuse a packet
func (n *Node) Diffuse(p *handel.Packet) {
	gossip := &Gossip{
		Topic: handelTopic,
		ID:    p.Origin,
		Msg:   p.MultiSig,
	}
	n.Gossip(gossip)
}

// Gossip gossips a message on the given topic by saving it into the right
// state- the state will then broadcasts it if not seen previously
func (n *Node) Gossip(g *Gossip) {
	var state = n.getOrCreate(g.Topic)
	state.NewGossip(g)
}

// Connect do nothing
func (n *Node) Connect(id handel.Identity) error {
	return nil
}

// NewPacket save the incoming gossip and reacts
func (n *Node) NewPacket(p *handel.Packet) {
	// decode a Gossip from the MultiSig packet
	dec := gob.NewDecoder(bytes.NewBuffer(p.MultiSig))
	gossip := &Gossip{}
	if err := dec.Decode(gossip); err != nil {
		fmt.Println(n.Identity().ID(), " - err:", err)
		return
	}

	state := n.getOrCreate(gossip.Topic)
	if state.NewGossip(gossip) {
		n.getTopicChannel(gossip.Topic) <- *gossip
	}
}

func (n *Node) getTopicChannel(topicID string) chan Gossip {
	n.Lock()
	defer n.Unlock()
	ch, exists := n.gossipsOut[topicID]
	if !exists {
		ch = make(chan Gossip, n.reg.Size())
		n.gossipsOut[topicID] = ch
	}
	return ch
}

func (n *Node) getOrCreate(topicID string) *topic {
	n.Lock()
	defer n.Unlock()
	state, exists := n.topics[topicID]
	if !exists {
		state = newTopic(n.Identity().ID(), topicID, n.reg.Size(), n.fanout, n.period, n.sendGossip)
		n.topics[topicID] = state
	}
	return state
}

// sendGossip marshals the gossip and sends it out to the given ID
func (n *Node) sendGossip(ids []int32, gossip *Gossip) {
	// marshal gossip into MultiSig of packet
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(gossip); err != nil {
		fmt.Println(n.Identity().ID(), " - err: ", err)
		return
	}
	p := &handel.Packet{
		Origin:   n.Identity().ID(),
		MultiSig: b.Bytes(),
	}

	identities := make([]handel.Identity, len(ids))
	for i := 0; i < len(ids); i++ {
		identities[i], _ = n.reg.Identity(int(ids[i]))
	}
	//fmt.Println(n.Identity().ID(), " --> sendGossip to ", ids)
	n.net.Send(identities, p)
}

// NextTopic returns the next channel where incoming gossip message for the
// given topic will be sent
func (n *Node) NextTopic(topicID string) chan Gossip {
	return n.getTopicChannel(topicID)
}

// Next re
func (n *Node) Next() chan handel.Packet {
	gossipChan := n.getTopicChannel(handelTopic)
	packetChannel := make(chan handel.Packet, cap(gossipChan))
	go func() {
		for g := range gossipChan {
			packetChannel <- handel.Packet{
				Origin:   g.ID,
				MultiSig: g.Msg,
			}
		}
	}()
	return packetChannel
}

// StopGossip stops the gossiping for the given topic
func (n *Node) StopGossip(topicID string) {
	n.getOrCreate(topicID).stop()
}

// Values implement the monitor.Counter interface
func (n *Node) Values() map[string]float64 {
	return n.counter.Values()
}
