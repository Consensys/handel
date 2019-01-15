package p2p

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
)

// Node is an interface to be used by an Aggregator
type Node interface {
	Identity() handel.Identity
	SecretKey() lib.SecretKey
	Diffuse(*handel.Packet)
	Connect(handel.Identity) error
	Next() chan handel.Packet
}

// Aggregator is a struct holding the logic to aggregates all signatures
// gossiped until it gets the final one
type Aggregator struct {
	Node
	sig       handel.Signature
	total     int
	threshold int
	rcvd      int
	out       chan *handel.MultiSignature
	r         handel.Registry
	accSig    handel.Signature
	accBs     handel.BitSet
	c         handel.Constructor
	acc       chan []byte
	done      chan bool
}

// NewAggregator returns an aggregator from the P2PNode
func NewAggregator(n Node, r handel.Registry, c handel.Constructor, sig handel.Signature, threshold int) *Aggregator {
	total := r.Size()
	return &Aggregator{
		Node:      n,
		sig:       sig,
		r:         r,
		rcvd:      0,
		c:         c,
		total:     total,
		threshold: threshold,
		accBs:     handel.NewWilffBitset(total),
		accSig:    c.Signature(),
		out:       make(chan *handel.MultiSignature, 1),
		acc:       make(chan []byte, total),
		done:      make(chan bool, 1),
	}
}

// FinalMultiSignature returns a channel that is used to signal the final
// multisignature is ready
func (a *Aggregator) FinalMultiSignature() chan *handel.MultiSignature {
	return a.out
}

// Start the aggregation for this node's perspective
func (a *Aggregator) Start() {
	ms := &handel.MultiSignature{
		Signature: a.sig,
		BitSet:    handel.NewWilffBitset(1),
	}
	msBuff, _ := ms.MarshalBinary()
	packet := &handel.Packet{
		Origin:   a.Identity().ID(),
		Level:    1, // just to always have same size as handel packets
		MultiSig: msBuff,
	}

	fmt.Printf("%d gossips signature %s\n", a.Node.Identity().ID(), hex.EncodeToString(msBuff[len(msBuff)-1-16:len(msBuff)-1]))
	a.Diffuse(packet)
	go a.handleIncoming()
}

func (a *Aggregator) handleIncoming() {
	for packet := range a.Node.Next() {
		//fmt.Printf("aggregator %d received packet from %d\n", a.P2PNode.handelID, packet.Origin)
		// check if already received
		if a.accBs.Get(int(packet.Origin)) {
			fmt.Println("already received - continue")
			continue
		}

		// verify it
		ms := new(handel.MultiSignature)
		err := ms.Unmarshal(packet.MultiSig, a.c.Signature(), handel.NewWilffBitset)
		if err != nil {
			panic(err)
		}

		id, ok := a.r.Identity(int(packet.Origin))
		if !ok {
			panic("some guy does not exist")
		}
		err = id.PublicKey().VerifySignature(lib.Message, ms.Signature)
		if err != nil {
			fmt.Printf("INVALID: %d verified signature from %d : %s\n", a.Node.Identity().ID(),
				packet.Origin, hex.EncodeToString(packet.MultiSig[0:16]))
			continue
		}
		// add it to the accumulator
		a.accSig = a.accSig.Combine(ms.Signature)
		a.accBs.Set(int(packet.Origin), true)
		a.rcvd++
		//fmt.Println(a.Node.Identity().ID(), " got sig from", packet.Origin, " -> ", a.rcvd, "/", a.total)
		// are we done ?
		if a.rcvd >= a.threshold {
			//fmt.Println("looping out")
			a.out <- &handel.MultiSignature{
				Signature: a.accSig,
				BitSet:    a.accBs,
			}
			break
		}
	}
}

// ConsFunc is a type to implement Constructor for a func
type ConsFunc func(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node)

// Make implements the Constructor interface
func (c *ConsFunc) Make(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node) {
	return (*c)(ctx, nodes, ids, opts)
}
