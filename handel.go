package handel

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// This struct keeps our state for all the levels we have. Most of the
//  time we will have multiple levels activated at the same time:
//    1) We will receive signatures for other peers
//    2) We will send signatures to other peers even if we have not finished the
//      previous levels
type level struct {
	// The id of this level. Start at 1
	id int

	// Our peers in this level: they send us their sigs, we're sending ours.
	nodes []Identity

	// True if we can start to send messages for this level.
	sendStarted bool

	// True is this level is completed for the reception, i.e. we have all the sigs
	rcvCompleted bool

	// We send updates to the peers, and we contact the peers one after the other
	// This field reference our current position in our list of peers.
	sendPos int

	// Count of peers contacted for the current sig
	// If we sent our current signature to all our peers we stop until we have
	//  a better signature for this level
	sendPeersCt int

	// Size of the current sig we're sending. This allows to check if we have a
	//  better signature.
	sendSigSize int
}

// newLevel returns a fresh new level at the given id (number) for these given
// nodes to contact.
func newLevel(id int, nodes []Identity) *level {
	if id <= 0 {
		panic("bad value for level id")
	}
	l := &level{
		id:           id,
		nodes:        nodes,
		sendStarted:  id == 1, // We can start the level 1 immediately: it's only our sig.
		rcvCompleted: false,
		sendPos:      0,
		sendPeersCt:  0,
		sendSigSize:  0,
	}
	return l
}

// Create a map of all the levels for this registry.
func createLevels(r Registry, partitioner Partitioner) map[int]*level {
	lvls := make(map[int]*level)

	for i := 1; i <= partitioner.MaxLevel(); i++ {
		nodes, _ := partitioner.PickNextAt(i, r.Size()+1)
		lvls[i] = newLevel(i, nodes)
	}

	return lvls
}

func (l *level) active() bool {
	return l.sendPeersCt < len(l.nodes) && l.sendStarted
}

// Select the peers we should contact next.
func (l *level) selectNextPeers(count int) ([]Identity, bool) {
	size := min(count, len(l.nodes))
	res := make([]Identity, size)

	for i := 0; i < size; i++ {
		res[i] = l.nodes[l.sendPos]
		l.sendPos++
		if l.sendPos >= len(l.nodes) {
			l.sendPos = 0
		}
	}

	l.sendPeersCt += size
	return res, true
}

// check if the signature is better than what we have.
// If it's better, reset the counters of the messages sent.
// If the level is now rcvCompleted we return true; if not we return false
func (l *level) updateSigToSend(sig *MultiSignature) bool {
	if l.sendSigSize >= sig.Cardinality() {
		return false
	}

	l.sendSigSize = sig.Cardinality()
	l.sendPeersCt = 0

	if l.sendSigSize == len(l.nodes) {
		// If we have all the signatures to send
		//  we can start the level without waiting for the timeout
		l.sendStarted = true
		return true
	}
	return false
}

// Send our best signature set for this level, to 'count' nodes
func (h *Handel) sendUpdate(l *level, count int) {
	if !l.active() {
		panic("level not started!")
	}

	sp := h.store.Combined(byte(l.id) - 1)
	newNodes, _ := l.selectNextPeers(count)
	h.sendTo(l.id, sp, newNodes)
}

// HStats contain minimal stats about handel
type HStats struct {
	msgSentCt int
	msgRcvCt  int
}

// Handel is the principal struct that performs the large scale multi-signature
// aggregation protocol. Handel is thread-safe.
type Handel struct {
	sync.Mutex
	stats HStats
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
	// signature store with different merging/caching strategy
	store signatureStore
	// processing of signature - verification strategy
	proc signatureProcessing
	// all actors registered that acts on a new signature
	actors []actor
	// best final signature,i.e. at the last level, seen so far
	best *MultiSignature
	// channel to exposes multi-signatures to the user
	out chan MultiSignature
	// indicating whether handel is finished or not
	done bool
	// constant threshold of contributions required in a ms to be considered
	// valid
	threshold int
	// ticker for the periodic update
	ticker *time.Ticker
	// all the levels
	levels map[int]*level
	// Start time of Handel. Used to calculate the timeouts
	startTime time.Time
}

// NewHandel returns a Handle interface that uses the given network and
// registry. The identity is the public identity of this Handel's node. The
// constructor defines over which curves / signature scheme Handel runs. The
// message is the message to "multi-sign" by Handel.  The first config in the
// slice is taken if not nil. Otherwise, the default config generated by
// DefaultConfig() is used.
func NewHandel(n Network, r Registry, id Identity, c Constructor,
	msg []byte, s Signature, conf ...*Config) *Handel {

	var config *Config
	if len(conf) > 0 && conf[0] != nil {
		config = mergeWithDefault(conf[0], r.Size())
	} else {
		config = DefaultConfig(r.Size())
	}

	part := config.NewPartitioner(id.ID(), r)
	firstBs := config.NewBitSet(1)
	firstBs.Set(0, true)
	mySig := &MultiSignature{BitSet: firstBs, Signature: s}

	h := &Handel{
		c:      config,
		net:    n,
		reg:    r,
		id:     id,
		cons:   c,
		msg:    msg,
		sig:    s,
		out:    make(chan MultiSignature, 10000),
		ticker: time.NewTicker(config.UpdatePeriod),
		levels: createLevels(r, part),
	}
	h.actors = []actor{
		actorFunc(h.checkCompletedLevel),
		actorFunc(h.checkFinalSignature),
	}

	h.threshold = h.c.ContributionsThreshold(h.reg.Size())
	h.store = newReplaceStore(part, h.c.NewBitSet, c)
	h.store.Store(0, mySig) // Our own sig is at level 0.
	// TODO change that to config item
	evaluator := h.c.EvaluatorStrategy(h.store, h)
	h.proc = newEvaluatorProcessing(part, c, msg, evaluator, h)
	h.net.RegisterListener(h)
	return h
}

// NewPacket implements the Listener interface for the network.
// it parses the packet and sends it to processing if the packet is properly
// formatted.
func (h *Handel) NewPacket(p *Packet) {
	h.Lock()
	defer h.Unlock()

	if h.done {
		return
	}
	ms, err := h.parsePacket(p)
	if err != nil {
		h.logf("invalid packet: %s", err)
		return
	}

	// sends it to processing
	if !h.getLevel(p.Level).rcvCompleted {
		msg := fmt.Sprintf("packet received from %d for level %d", p.Origin, p.Level)
		h.logf(msg)

		//h.logf("%s - done ", msg)
		h.proc.Add(&sigPair{origin: p.Origin, level: p.Level, ms: ms})
	}
}

// Start the Handel protocol by sending signatures to peers in the first level,
// and by starting relevant sub routines.
func (h *Handel) Start() {
	h.Lock()
	defer h.Unlock()
	h.startTime = time.Now()
	go h.proc.Start()
	go h.rangeOnVerified()
	go h.periodicLoop()
	h.periodicUpdate()
}

func (h *Handel) periodicLoop() {
	for range h.ticker.C {
		h.Lock()
		h.periodicUpdate()
		h.Unlock()
	}
}

// Stop the Handel protocol and all sub routines
func (h *Handel) Stop() {
	h.Lock()
	defer h.Unlock()
	h.ticker.Stop()
	h.proc.Stop()
	h.done = true
	close(h.out)
}

// Does the periodic update:
//  - check if we reached a timeout for each level
//  - send a new packet
// You must have locked handel before calling this function
func (h *Handel) periodicUpdate() {
	for i := byte(1); i <= byte(len(h.levels)); i++ {
		lvl := h.getLevel(i)
		if !lvl.sendStarted {
			h.decideToStartLevel(lvl)
		}
		if lvl.active() {
			h.sendUpdate(lvl, 1)
		}
	}
}

func (h *Handel) decideToStartLevel(l *level) {
	msSinceStart := int(time.Now().Sub(h.startTime).Seconds() * 1000)
	if msSinceStart >= l.id*int(h.c.LevelTimeout.Seconds())*1000 {
		l.sendStarted = true
	}
}

// FinalSignatures returns the channel over which final multi-signatures
// are sent over. These multi-signatures contain at least a threshold of
// contributions, as defined in the config.
func (h *Handel) FinalSignatures() chan MultiSignature {
	return h.out
}

// rangeOnVerified continuously listens on the output channel of the signature
// processing routine for verified signatures. Each verified signatures is
//  1) Added to the store of verified signature
//  2) passed down to all registered actors. Each handler is called in a thread safe
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

// checkFinalSignature checks if anew better final signature (ig. a signature at the last level) has been
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

	newCard := sig.Cardinality()
	local := h.best.Cardinality()
	if newCard > local {
		newBest(sig)
	}
}

func (h *Handel) getLevel(levelID byte) *level {
	l := int(levelID)
	if l <= 0 || l > len(h.levels) {
		msg := fmt.Sprintf("Bad level (%d) max is %d", l, len(h.levels))
		panic(msg)
	}
	return h.levels[l]
}

// When we have a new signature, multiple levels may be impacted. The store
//  is in charge of selecting the best signature for a level, so we will
//  call it for all possibly impacted levels.
func (h *Handel) checkCompletedLevel(s *sigPair) {
	// The receiving phase: have we completed this level?
	lvl := h.getLevel(s.level)
	if s.ms.Cardinality() == len(lvl.nodes) {
		lvl.rcvCompleted = true
	}

	// The sending phase: for all upper levels we may have completed
	//  the level. We check & send an update if it's the case
	for i := s.level + 1; i <= byte(len(h.levels)); i++ {
		lvl := h.getLevel(i)
		ms := h.store.Combined(byte(lvl.id) - 1)
		if ms != nil && lvl.updateSigToSend(ms) {
			h.sendUpdate(lvl, h.c.CandidateCount)
		}
	}
}

func (h *Handel) sendTo(lvl int, ms *MultiSignature, ids []Identity) {
	h.stats.msgSentCt++

	buff, err := ms.MarshalBinary()
	if err != nil {
		h.logf("error marshalling multi-signature: %s", err)
		return
	}

	p := &Packet{
		Origin:   h.id.ID(),
		Level:    byte(lvl),
		MultiSig: buff,
	}

	msg := fmt.Sprintf("packet sent of level %d to %v -- %s", p.Level, ids, h.store)
	h.logf(msg)
	h.net.Send(ids, p)
}

// parsePacket returns the multisignature parsed from the given packet, or an
// error if the packet can't be unmarshalled, or contains erroneous data such as
// out of range level.  This method is NOT thread-safe and only meant for
// internal use.
func (h *Handel) parsePacket(p *Packet) (*MultiSignature, error) {
	h.stats.msgRcvCt++

	if p.Origin < 0 || p.Origin >= int32(h.reg.Size()) {
		return nil, errors.New("packet's origin out of range")
	}

	lvl := int(p.Level)
	if lvl < 1 || lvl > log2(h.reg.Size()) {
		msg := fmt.Sprintf("packet's level out of range, level received=%d, max=%d, nodes count=%d",
			lvl, log2(h.reg.Size()), h.reg.Size())
		return nil, errors.New(msg)
	}

	ms := new(MultiSignature)
	err := ms.Unmarshal(p.MultiSig, h.cons.Signature(), h.c.NewBitSet)
	return ms, err
}

func (h *Handel) logf(str string, args ...interface{}) {
	now := time.Now()
	timeSpent := fmt.Sprintf("%02d:%02d:%02d", now.Hour(),
		now.Minute(),
		now.Second())
	idArg := []interface{}{timeSpent, h.id.ID()}
	logf("%s: handel %d: "+str, append(idArg, args...)...)
}
