package p2p

import (
	"container/list"
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

// Node is an interface to be used by an Aggregator
type Node interface {
	monitor.Counter
	Identity() handel.Identity
	SecretKey() lib.SecretKey
	Diffuse(*handel.Packet)
	Connect(handel.Identity) error
	Next() chan handel.Packet
}

// Aggregator is a struct holding the logic to aggregates all signatures
// gossiped until it gets the final one
type Aggregator struct {
	sync.Mutex
	Node
	sig          handel.Signature
	total        int
	threshold    int
	rcvd         int
	out          chan *handel.MultiSignature
	r            handel.Registry
	accSig       handel.Signature
	accBs        handel.BitSet
	c            handel.Constructor
	acc          chan []byte
	done         chan bool
	resendP      time.Duration
	tick         *time.Ticker
	ctx          context.Context
	procReady    chan bool
	newPacket    chan handel.Packet
	aggAndVerify bool
	// only used when aggAndVerify is true - aggregate everything then search
	// for the bad ones
	indSigs map[int]handel.Signature
}

// NewAggregator returns an aggregator from the P2PNode
func NewAggregator(ctx context.Context, n Node, r handel.Registry, c handel.Constructor, sig handel.Signature, threshold int, resendPeriod time.Duration, aggAndVerify bool) *Aggregator {
	total := r.Size()
	return &Aggregator{
		aggAndVerify: aggAndVerify,
		Node:         n,
		sig:          sig,
		r:            r,
		rcvd:         0,
		c:            c,
		total:        total,
		threshold:    threshold,
		accBs:        handel.NewWilffBitset(total),
		accSig:       c.Signature(),
		out:          make(chan *handel.MultiSignature, 1),
		acc:          make(chan []byte, total),
		done:         make(chan bool, 1),
		procReady:    make(chan bool, 1),
		newPacket:    make(chan handel.Packet, 10),
		resendP:      resendPeriod,
		ctx:          ctx,
		indSigs:      make(map[int]handel.Signature),
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

	a.tick = time.NewTicker(a.resendP)
	go func() {
		// diffuse it right away once
		//fmt.Printf("%d gossips signature %s - pk = %s\n", a.Node.Identity().ID(), hex.EncodeToString(msBuff[len(msBuff)-1-16:len(msBuff)-1]), a.Identity().PublicKey().String())
		a.Diffuse(packet)
		for {
			select {
			case <-a.tick.C:
				a.Diffuse(packet)
				fmt.Printf("%d gossips signature %s\n", a.Node.Identity().ID(), hex.EncodeToString(msBuff[len(msBuff)-1-16:len(msBuff)-1]))
			case <-a.ctx.Done():
				return
			case <-a.done:
				return
			}
		}
	}()
	go a.handleIncoming()
	go a.readNexts()
}

func (a *Aggregator) readNexts() {
	defer close(a.newPacket)
	packets := list.New()
	var ready bool
	send := func() {
		if packets.Len() == 0 {
			ready = true
			return
		}
		p := packets.Remove(packets.Front()).(handel.Packet)
		a.newPacket <- p
		ready = false
	}
	for {
		select {
		case packet := <-a.Node.Next():
			if len(packet.MultiSig) == 0 {
				continue
			}
			packets.PushBack(packet)
			if ready {
				send()
			}
		case <-a.procReady:
			send()
		case <-a.ctx.Done():
			return
		case <-a.done:
			return
		}
	}
}

func (a *Aggregator) handleIncoming() {
	a.procReady <- true
	for {
		select {
		case <-a.done:
			return
		case packet := <-a.newPacket:
			if a.aggAndVerify {
				a.aggregate(packet)
			} else {
				a.verifyPacket(packet)
			}
		}
	}
}

func (a *Aggregator) aggregate(packet handel.Packet) {
	defer func() { a.procReady <- true }()
	if a.accBs.Get(int(packet.Origin)) {
		fmt.Println("already received - continue")
		return
	}

	ms := new(handel.MultiSignature)
	err := ms.Unmarshal(packet.MultiSig, a.c.Signature(), handel.NewWilffBitset)
	if err != nil {
		panic(err)
	}

	a.Lock()
	defer a.Unlock()
	// simply store it  and aggrregate it even if not verified yet
	a.indSigs[int(packet.Origin)] = ms.Signature
	a.accSig = a.accSig.Combine(ms.Signature)
	a.accBs.Set(int(packet.Origin), true)
	a.rcvd++
	fmt.Println(a.Node.Identity().ID(), "got sig from", packet.Origin, " -> ", a.rcvd, "/", a.total)
	// are we done
	if a.rcvd >= a.threshold {
		go a.verifyAndDispatch()
	}
}

// verify the aggregated signature first, then do a binary lookover if not
// correct to find the invalid ones then dispatch
func (a *Aggregator) verifyAndDispatch() {
	a.Lock()
	defer a.Unlock()
	ms := &handel.MultiSignature{
		Signature: a.accSig,
		BitSet:    a.accBs,
	}
	err := handel.VerifyMultiSignature(lib.Message, ms, a.r, a.c)
	if err != nil {
		//panic(" --- aggregator invalid !!")
		// TODO BINARY SEARCH
		return
	}
	fmt.Println(a.Identity().ID(), " -- Dispatched Signature -- ")
	copySig := a.c.Signature()
	buff, _ := a.accSig.MarshalBinary()
	if err := copySig.UnmarshalBinary(buff); err != nil {
		panic(err)
	}
	a.out <- &handel.MultiSignature{
		Signature: copySig,
		BitSet:    a.accBs.Clone(),
	}
	//close(a.done)
	return
}

func (a *Aggregator) verifyPacket(packet handel.Packet) {
	defer func() { a.procReady <- true }()
	//fmt.Printf("aggregator %d received packet from %d\n", a.P2PNode.handelID, packet.Origin)
	// check if already received
	if a.accBs.Get(int(packet.Origin)) {
		fmt.Println("already received - continue")
		return
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
		msBuff := packet.MultiSig
		fmt.Printf("INVALID: %d verified signature from %d : %s - pk = %s\n", a.Node.Identity().ID(),
			packet.Origin, hex.EncodeToString(msBuff[len(msBuff)-1-16:len(msBuff)-1]), id.PublicKey().String())

		return
	}
	// add it to the accumulator
	a.accSig = a.accSig.Combine(ms.Signature)
	a.accBs.Set(int(packet.Origin), true)
	a.rcvd++
	fmt.Println(a.Node.Identity().ID(), " got sig from", packet.Origin, " -> ", a.rcvd, "/", a.total)
	// are we done ?
	if a.rcvd >= a.threshold {
		//fmt.Println("looping OUUUTTTT")
		a.out <- &handel.MultiSignature{
			Signature: a.accSig,
			BitSet:    a.accBs,
		}
		close(a.done)
		return
	}
}

// ConsFunc is a type to implement Constructor for a func
type ConsFunc func(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node)

// Make implements the Constructor interface
func (c *ConsFunc) Make(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node) {
	return (*c)(ctx, nodes, ids, opts)
}
