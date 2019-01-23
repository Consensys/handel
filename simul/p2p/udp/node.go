package main

import (
	"fmt"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/ConsenSys/handel/network/udp"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
	"github.com/ConsenSys/handel/simul/p2p"
)

var _ p2p.Node = (*Node)(nil)

// Node implements the p2p.Node interface using UDP
type Node struct {
	handel.Network
	sec lib.SecretKey
	id  handel.Identity
	reg handel.Registry
	out chan handel.Packet
}

// NewNode returns a UDP based node
func NewNode(sec lib.SecretKey, id handel.Identity, reg handel.Registry, enc network.Encoding) *Node {
	net, err := udp.NewNetwork(id.Address(), enc)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	n := &Node{
		sec:     sec,
		id:      id,
		reg:     reg,
		Network: net,
		out:     make(chan handel.Packet, reg.Size()),
	}
	n.Network.RegisterListener(n)
	return n
}

// SecretKey implements the p2p.Node interface
func (n *Node) SecretKey() lib.SecretKey {
	return n.sec
}

// Identity implements the p2p.Node interface
func (n *Node) Identity() handel.Identity {
	return n.id
}

// Diffuse implements the p2p.Node interface
func (n *Node) Diffuse(p *handel.Packet) {
	max := n.reg.Size()
	ids, ok := n.reg.Identities(0, max)
	if !ok {
		fmt.Println("can't diffuse UDP packet")
		panic("aie")
	}
	n.Network.Send(ids, p)
}

// Next implements the p2p.Node interface
func (n *Node) Next() chan handel.Packet {
	return n.out
}

// NewPacket implements the p2p.Node interface
func (n *Node) NewPacket(p *handel.Packet) {
	n.out <- *p
}

// Connect implements the p2p.Node interface
func (n *Node) Connect(id handel.Identity) error {
	// no connection with UDP
	return nil
}

// Values implement the monitor.Counter interface
func (n *Node) Values() map[string]float64 {
	return n.Network.(monitor.Counter).Values()
}
