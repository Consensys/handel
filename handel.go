package handel

import (
	"errors"
	"fmt"
	"sync"
)

// Handel is the principal struct that performs the large scale multi-signature
// aggregation protocol. Handel is thread-safe.
type Handel struct {
	sync.Mutex
	// Config holding parameters to Handel
	c *Config
	// Network enabling external communication with other Handel nodes
	net Network
	// Registry holding access to all Handel node's identities
	reg Registry
	// constructor to unmarshal signatures + aggregate pub keys
	cons Constructor
	// public identity of this Handel node
	id Identity
	// Message that is being signed during the Handel protocol
	msg []byte
	// signature over the message
	sig Signature
	// partitions the set of nodes at different levels
	part partitioner
	// signature store with different merging/caching strategy
	store signatureStore
	// processing of signature - verification strategy
	proc signatureProcessing
	// all actors registered that acts on a new signature
	actors []actor
	// completed levels, i.e. full signatures at each of these levels
	completed []byte
	// highest level attained by this handel node so far
	currLevel byte
	// maximum  level attainable ever for this set of nodes
	maxLevel byte
	// best final signature,i.e. at the last level, seen so far
	best *MultiSignature
	// channel to exposes multi-signatures to the user
	out chan MultiSignature
	// indicating whether handel is finished or not
	done bool
	// constant threshold of contributions required in a ms to be considered
	// valid
	threshold int
}

// NewHandel returns a Handle interface that uses the given network and
// registry. The identity is the public identity of this Handel's node. The
// constructor defines over which curves / signature scheme Handel runs. The
// message is the message to "multi-sign" by Handel.  The first config in the
// slice is taken if not nil. Otherwise, the default config generated by
// DefaultConfig() is used.
func NewHandel(n Network, r Registry, id Identity, c Constructor,
	msg []byte, s Signature, conf ...*Config) *Handel {
	h := &Handel{
		net:      n,
		reg:      r,
		part:     newBinTreePartition(id.ID(), r),
		id:       id,
		cons:     c,
		msg:      msg,
		sig:      s,
		maxLevel: byte(log2(r.Size())),
		out:      make(chan MultiSignature, 100),
	}
	h.actors = []actor{
		actorFunc(h.checkCompletedLevel),
		actorFunc(h.checkFinalSignature),
	}

	if len(conf) > 0 && conf[0] != nil {
		h.c = mergeWithDefault(conf[0], r.Size())
	} else {
		h.c = DefaultConfig(r.Size())
	}

	h.threshold = h.c.ContributionsThreshold(h.reg.Size())
	h.store = newReplaceStore(h.part, h.c.NewBitSet)
	firstBs := h.c.NewBitSet(1)
	firstBs.Set(0, true)
	h.store.Store(0, &MultiSignature{BitSet: firstBs, Signature: s})
	h.proc = newFifoProcessing(h.store, h.part, c, msg)
	h.net.RegisterListener(h)
	return h
}

// NewPacket implements the Listener interface for the network.
// it parses the packet and sends it to processing if the packet is properly
// formatted.
func (h *Handel) NewPacket(p *Packet) {
	h.Lock()
	defer h.Unlock()
	ms, err := h.parsePacket(p)
	if err != nil {
		h.logf(err.Error())
	}

	// sends it to processing
	h.logf("sending incoming signature from %d to verification thread", p.Origin)
	h.proc.Incoming() <- sigPair{level: p.Level, ms: ms}
}

// Start the Handel protocol by sending signatures to peers in the first level,
// and by starting relevant sub routines.
func (h *Handel) Start() {
	h.Lock()
	defer h.Unlock()
	go h.proc.Start()
	go h.rangeOnVerified()
	h.startNextLevel()
}

// Stop the Handel protocol and all sub routines
func (h *Handel) Stop() {
	h.Lock()
	defer h.Unlock()
	h.proc.Stop()
	h.done = true
	close(h.out)
}

// FinalSignatures returns the channel over which final multi-signatures
// are sent over. These multi-signatures contain at least a threshold of
// contributions, as defined in the config.
func (h *Handel) FinalSignatures() chan MultiSignature {
	return h.out
}

// parsePacket returns the multisignature parsed from the given packet, or an
// error if the packet can't be unmarshalled, or contains erroneous data such as
// out of range level.  This method is NOT thread-safe and only meant for
// internal use.
func (h *Handel) parsePacket(p *Packet) (*MultiSignature, error) {
	if p.Origin >= int32(h.reg.Size()) {
		return nil, errors.New("handel: packet's origin out of range")
	}

	if int(p.Level) > log2(h.reg.Size()) {
		return nil, errors.New("handel: packet's level out of range")
	}

	ms := new(MultiSignature)
	err := ms.Unmarshal(p.MultiSig, h.cons.Signature(), h.c.NewBitSet)
	return ms, err
}

// startNextLevel increase the currLevel counter and sends its best
// highest-level signature it has to nodes at the new currLevel.
func (h *Handel) startNextLevel() {
	if h.currLevel >= h.maxLevel {
		// protocol is finished
		h.logf("protocol finished at level %d", h.currLevel)
		return
	}
	h.currLevel++
	sp := h.store.Highest()
	if sp == nil {
		h.logf("no signature to send ...?")
		return
	}
	nodes, ok := h.part.PickNextAt(int(h.currLevel), h.c.CandidateCount)
	if !ok {
		// XXX This should not happen, but what if ?
		return
	}
	// NOTE: send with the actual level of the multisignature for the size of the
	// bitset is correct and verification will pass.
	// XXX: either put the +1 when doing the BestCombined or just here. +1 is
	// needed since when aggregating different-level signature, you are
	// generating the signature for the next level.
	h.sendTo(sp.level+1, sp.ms, nodes)
	fmt.Printf(" --- NEW LEVEL send sigpair: %+v\n", sp)
	fmt.Printf(" ---- replaceStore string(): %s\n", h.store)
	h.logf("new level %d: sent best signatures (lvl = %d) to %d nodes", h.currLevel, sp.level+1, len(nodes))
}

// rangeOnVerified continuously listens on the output channel of the signature
// processing routine for verified signatures. Each verified signatures is
// passed down to all registered actors. Each handler is called in a thread safe
// manner, global lock is held during the call to actors.
func (h *Handel) rangeOnVerified() {
	for v := range h.proc.Verified() {
		h.store.Store(v.level, v.ms)
		h.Lock()
		for _, actor := range h.actors {
			actor.OnVerifiedSignature(&v)
		}
		h.Unlock()
	}
}

// actor is an interface that takes a new verified signature and acts on it
// according to its own rule. It can be checking if it passes to a next level,
// checking if the protocol is finished, checking if a signature completes
// higher levels so it should send it out to other peers, etc. The store is
// guaranteed to have a multisignature present at the level indicated in the
// verifiedSig. Each handler is called in a thread safe manner, global lock is
// held during the call to actors.
type actor interface {
	OnVerifiedSignature(s *sigPair)
}

type actorFunc func(s *sigPair)

func (a actorFunc) OnVerifiedSignature(s *sigPair) {
	a(s)
}

// checkFinalSignature STORES the newly verified signature and then checks if a
// new better final signature, i.e. a signature at the last level, has been
// generated. If so, it sends it to the output channel.
func (h *Handel) checkFinalSignature(s *sigPair) {
	sig := h.store.FullSignature()

	if sig.BitSet.Cardinality() < h.threshold {
		return
	}

	newBest := func(ms *MultiSignature) {
		if h.done {
			return
		}
		h.best = ms
		h.out <- *h.best
	}

	if h.best == nil {
		newBest(sig)
		return
	}

	new := sig.Cardinality()
	local := h.best.Cardinality()
	if new > local {
		newBest(sig)
	}
}

// checNewLevel looks if the signature completes its respective level. If it
// does, handel sends it out to new peers for this level if possible.
func (h *Handel) checkCompletedLevel(s *sigPair) {
	if h.isCompleted(s.level) {
		return
	}

	// XXX IIF completed signatures for higher level then send this higher level
	// instead
	ms, ok := h.store.Best(s.level)
	if !ok {
		panic("something's wrong with the store")
	}
	fullSize, err := h.part.Size(int(s.level))
	if err != nil {
		panic("level should be verified before")
	}
	if s.ms.Cardinality() != fullSize {
		fmt.Println(" signature NOT FULL ??????????")
		fmt.Println("ms.Car() ", s.ms.Cardinality(), " vs fullSize ", fullSize)
		return
	}

	// completed level !
	h.markCompleted(s.level)

	// TODO: if no new nodes are available, maybe send to same nodes again
	// in case for full signatures ?
	newNodes, ok := h.part.PickNextAt(int(s.level), h.c.CandidateCount)
	if ok {
		h.logf("sending complete signature for level %d to %d new nodes", s.level, len(newNodes))
		h.sendTo(s.level, ms, newNodes)
	} else {
		h.logf("no new nodes for completed level %d", s.level)
	}

	// go to next level if we already finished this one !
	if s.level == h.currLevel {
		go h.startNextLevel()
	}
}

func (h *Handel) sendTo(lvl byte, ms *MultiSignature, ids []Identity) {
	buff, err := ms.MarshalBinary()
	if err != nil {
		h.logf("error marshalling multi-signature: %s", err)
		return
	}

	packet := &Packet{
		Origin:   h.id.ID(),
		Level:    lvl,
		MultiSig: buff,
	}
	h.net.Send(ids, packet)
}

// isCompleted returns true if the given level has already been completed, i.e.
// is in the list of completed levels.
func (h *Handel) isCompleted(level byte) bool {
	for _, l := range h.completed {
		if l == level {
			return true
		}
	}
	return false
}

func (h *Handel) markCompleted(level byte) {
	h.completed = append(h.completed, level)
}

func (h *Handel) logf(str string, args ...interface{}) {
	idArg := []interface{}{h.id.ID()}
	logf("handel %d: "+str, append(idArg, args...)...)
}
