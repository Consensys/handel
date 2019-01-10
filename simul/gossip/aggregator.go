package main

import (
	"fmt"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
)

// Aggregator is a struct holding the logic to aggregates all signatures
// gossiped until it gets the final one
type Aggregator struct {
	*P2PNode
	total  int
	rcvd   int
	out    chan *handel.MultiSignature
	r      handel.Registry
	accSig handel.Signature
	accBs  handel.BitSet
	c      handel.Constructor
	acc    chan []byte
	done   chan bool
}

// NewAggregator returns an aggregator from the P2PNode
func NewAggregator(n *P2PNode, r handel.Registry, c handel.Constructor, total int) *Aggregator {
	return &Aggregator{
		P2PNode: n,
		r:       r,
		total:   total,
		rcvd:    0,
		c:       c,
		accBs:   handel.NewWilffBitset(total),
		accSig:  c.Signature(),
		out:     make(chan *handel.MultiSignature, 1),
		acc:     make(chan []byte, total),
		done:    make(chan bool, 1),
	}
}

// FinalMultiSignature returns a channel that is used to signal the final
// multisignature is ready
func (a *Aggregator) FinalMultiSignature() chan *handel.MultiSignature {
	return a.out
}

// Start the aggregation for this node's perspective
func (a *Aggregator) Start() {
	sig, err := a.P2PNode.priv.SecretKey.Sign(lib.Message, nil)
	if err != nil {
		panic(err)
	}
	ms := &handel.MultiSignature{
		Signature: sig,
		BitSet:    handel.NewWilffBitset(1),
	}
	msBuff, _ := ms.MarshalBinary()
	packet := handel.Packet{
		Origin:   a.handelID,
		Level:    1, // just to always have same size as handel packets
		MultiSig: msBuff,
	}

	buff, err := packet.MarshalBinary()
	if err != nil {
		panic(err)
	}

	if err := a.Gossip(buff); err != nil {
		panic(err)
	}
	go a.handleIncoming()
	go a.processLoop()
}

func (a *Aggregator) handleIncoming() {
	for {
		buff, err := a.Next()
		if err != nil {
			fmt.Println("error !!", err)
			break
		}
		a.acc <- buff
	}
}

func (a *Aggregator) processLoop() {
	for {
		var buff []byte
		select {
		case buff = <-a.acc:
		case <-a.done:
			return
		}
		packet := new(handel.Packet)
		if err := packet.UnmarshalBinary(buff); err != nil {
			fmt.Println("error unmarshaling ", err)
			panic(err)
		}
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
			panic(err)
		}
		// add it to the accumulator
		a.accSig = a.accSig.Combine(ms.Signature)
		a.accBs.Set(int(packet.Origin), true)
		a.rcvd++
		//fmt.Println(a.handelID, " got sig from", packet.Origin, " -> ", a.rcvd, "/", a.total)
		// are we done ?
		if a.rcvd == a.total {
			//fmt.Println("looping out")
			a.out <- &handel.MultiSignature{
				Signature: a.accSig,
				BitSet:    a.accBs,
			}
			break
		}

	}
}
