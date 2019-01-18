package handel

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	// The size of the signature we send at this level. It's not symmetric if
	//  we don't have a power of two for the numbers of nodes: we may have a number of
	//  signatures to send greater (or smaller!) than the number of peers we have
	//  at this level
	sendExpectedFullSize int

	// Size of the current sig we're sending. This allows to check if we have a
	//  better signature.
	sendSigSize int
}

// newLevel returns a fresh new level at the given id (number) for these given
// nodes to contact.
func newLevel(id int, nodes []Identity, sendExpectedFullSize int) *level {
	if id <= 0 {
		panic("bad value for level id")
	}
	l := &level{
		id:                   id,
		nodes:                nodes,
		sendStarted:          false,
		rcvCompleted:         false,
		sendPos:              0,
		sendPeersCt:          0,
		sendExpectedFullSize: sendExpectedFullSize,
		sendSigSize:          0,
	}
	return l
}

// Create a map of all the levels for this registry.
func createLevels(c *Config, partitioner Partitioner) map[int]*level {
	lvls := make(map[int]*level)
	var firstActive bool
	sendExpectedFullSize := 1
	for _, level := range partitioner.Levels() {
		nodes2, _ := partitioner.IdentitiesAt(level)
		nodes := nodes2
		if !c.DisableShuffling {
			nodes = make([]Identity, len(nodes2))
			copy(nodes, nodes2)
			shuffle(nodes, c.Rand)
		}
		lvls[level] = newLevel(level, nodes, sendExpectedFullSize)
		sendExpectedFullSize += len(nodes)
		if !firstActive {
			lvls[level].setStarted()
			firstActive = true
		}
	}

	return lvls
}

func (l *level) active() bool {
	return l.started() && l.sendPeersCt < len(l.nodes)
}

func (l *level) started() bool {
	return l.sendStarted
}

func (l *level) setStarted() {
	l.sendStarted = true
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
// If the level is now complete, we return true; if not we return false
func (l *level) updateSigToSend(sig *MultiSignature) bool {
	if l.sendSigSize >= sig.Cardinality() {
		return false
	}

	l.sendSigSize = sig.Cardinality()
	l.sendPeersCt = 0

	if l.sendSigSize == l.sendExpectedFullSize {
		// If we have all the signatures to send
		// we can start the level without waiting for the timeout
		l.setStarted()
		return true
	}
	return false
}

func (l *level) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "level %d:", l.id)
	var nodes []string
	for _, n := range l.nodes {
		nodes = append(nodes, strconv.Itoa(int(n.ID())))
	}
	fmt.Fprintf(&b, "\t%s\n", strings.Join(nodes, ", "))
	return b.String()
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
	// Partitioning strategy used by the Handel round
	Partitioner Partitioner
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
	// ids of the level in order as returned by the partitioner
	ids []int
	// Start time of Handel. Used to calculate the timeouts
	startTime time.Time
	// the timeout strategy used by handel
	timeout TimeoutStrategy

	// the logger used by this Handel - always contains the ID
	log Logger
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
		c:           config,
		net:         n,
		reg:         r,
		Partitioner: part,
		id:          id,
		cons:        c,
		msg:         msg,
		sig:         s,
		out:         make(chan MultiSignature, 10000),
		ticker:      time.NewTicker(config.UpdatePeriod),
		log:         config.Logger.With("id", id.ID()),
		levels:      createLevels(config, part),
		ids:         part.Levels(),
	}
	h.actors = []actor{
		actorFunc(h.checkCompletedLevel),
		actorFunc(h.checkFinalSignature),
	}

	h.threshold = h.c.Contributions
	h.store = newReplaceStore(part, h.c.NewBitSet, c)
	h.store.Store(0, mySig) // Our own sig is at level 0.
	evaluator := h.c.NewEvaluatorStrategy(h.store, h)
	h.proc = newEvaluatorProcessing(part, c, msg, config.UnsafeSleepTimeOnSigVerify, evaluator, h.log)
	h.net.RegisterListener(h)
	h.timeout = h.c.NewTimeoutStrategy(h, h.ids)
	return h
}

// NewPacket implements the Listener interface for the network.
// it parses the packet and sends the multisignature if correct and the
// individual signature if correct.
func (h *Handel) NewPacket(p *Packet) {
	h.Lock()
	defer h.Unlock()

	if h.done {
		return
	}
	if err := h.validatePacket(p); err != nil {
		h.log.Warn("invalid_packet", err)
		return
	}
	ms, ind, err := h.parseSignatures(p)
	if err != nil {
		h.log.Warn("invalid_packet", err)
	} else if !h.getLevel(p.Level).rcvCompleted {
		// sends it to processing
		h.log.Debug("rcvd_from", p.Origin, "rcvd_level", p.Level)
		h.proc.Add(ms)
		if ind != nil {
			// can happen since we dont always send individual signature if this
			// is a complete level
			h.proc.Add(ind)
		}
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
	go h.timeout.Start()
	go h.periodicLoop()
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
	h.timeout.Stop()
	h.proc.Stop()
	h.done = true
	close(h.out)
}

// Does the periodic update:
//  - check if we reached a timeout for each level
//  - send a new packet
// You must have locked handel before calling this function
func (h *Handel) periodicUpdate() {
	for _, lvl := range h.levels {
		if lvl.active() {
			h.sendUpdate(lvl, h.c.UpdateCount)
		}
	}
}

// StartLevel starts the given level if not started already. It sends
// our best signature for this level up to CandidateCount peers.
func (h *Handel) StartLevel(level int) {
	h.Lock()
	defer h.Unlock()
	lvl := h.getLevel(byte(level))
	h.unsafeStartLevel(lvl)
}

func (h *Handel) unsafeStartLevel(lvl *level) {
	if lvl.started() {
		return
	}
	lvl.setStarted()
	h.sendUpdate(lvl, h.c.NodeCount)

}

// Send our best signature set for this level, to 'count' nodes. The level must
// be active before calling this method.
func (h *Handel) sendUpdate(l *level, count int) {
	sp := h.store.Combined(byte(l.id) - 1)
	newNodes, _ := l.selectNextPeers(count)
	h.sendTo(l.id, sp, newNodes)
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
	OnVerifiedSignature(s *incomingSig)
}

type actorFunc func(s *incomingSig)

func (a actorFunc) OnVerifiedSignature(s *incomingSig) {
	a(s)
}

// checkFinalSignature checks if anew better final signature (ig. a signature at the last level) has been
// generated. If so, it sends it to the output channel.
func (h *Handel) checkFinalSignature(s *incomingSig) {
	sig := h.store.FullSignature()

	if sig.BitSet.Cardinality() < h.threshold {
		return
	}
	newBest := func(ms *MultiSignature) {
		if h.done {
			return
		}
		h.best = ms
		h.log.Info("new_sig", fmt.Sprintf("%d/%d/%d", ms.Cardinality(), h.threshold, h.reg.Size()))
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
	lvl, exists := h.levels[l]
	if !exists {
		msg := fmt.Sprintf("inexistant level %d in list %v", l, h.ids)
		panic(msg)
	}
	return lvl
}

// When we have a new signature, multiple levels may be impacted. The store
//  is in charge of selecting the best signature for a level, so we will
//  call it for all possibly impacted levels.
func (h *Handel) checkCompletedLevel(s *incomingSig) {
	// The receiving phase: have we completed this level?
	lvl := h.getLevel(s.level)
	if lvl.rcvCompleted {
		return
	}

	sp, _ := h.store.Best(s.level)
	if sp == nil {
		panic("we should have received the best signature, we got nil!")
	}
	if sp.Cardinality() == len(lvl.nodes) {
		h.log.Debug("level_complete", s.level)
		lvl.rcvCompleted = true
	}

	// The sending phase: for all upper levels we may have completed the level.
	// We try to update all levels upwards & send an update if it's the case
	for id, lvl := range h.levels {
		if id < int(s.level+1) {
			continue
		}
		ms := h.store.Combined(byte(id) - 1)
		if ms != nil && lvl.updateSigToSend(ms) {
			h.sendUpdate(lvl, h.c.NodeCount)
		}
	}
}

func (h *Handel) sendTo(lvl int, ms *MultiSignature, ids []Identity) {
	h.stats.msgSentCt++

	buff, err := ms.MarshalBinary()
	if err != nil {
		h.log.Error("multi-signature", err)
		return
	}

	p := &Packet{
		Origin:   h.id.ID(),
		Level:    byte(lvl),
		MultiSig: buff,
	}

	h.log.Debug("sent_level", p.Level, "sent_nodes", fmt.Sprintf("%s", ids))
	h.net.Send(ids, p)
}

// validatePacket verifies the validity of the origin and level fields of the
// packet and returns an error if any.
func (h *Handel) validatePacket(p *Packet) error {
	h.stats.msgRcvCt++

	if p.Origin < 0 || p.Origin >= int32(h.reg.Size()) {
		return errors.New("packet's origin out of range")
	}

	_, exists := h.levels[int(p.Level)]

	if !exists {
		return fmt.Errorf("invalid packet's level %d", p.Level)
	}
	return nil
}

// parseMultisignature returns the multisignature unmarshalled if correct, or an
// error otherwise.
func (h *Handel) parseSignatures(p *Packet) (ms *incomingSig, ind *incomingSig, err error) {
	m := new(MultiSignature)
	err = m.Unmarshal(p.MultiSig, h.cons.Signature(), h.c.NewBitSet)
	if err != nil {
		return
	}

	// level is already check before
	lvl, _ := h.levels[int(p.Level)]
	if m.BitLength() != len(lvl.nodes) {
		err = errors.New("invalid bitset's size for given level")
		return
	}
	if m.None() {
		err = errors.New("no signature in the bitset")
		return
	}
	ms = &incomingSig{
		origin: p.Origin,
		level:  p.Level,
		ms:     m,
	}

	if p.IndividualSig == nil {
		return
	}
	individual := h.cons.Signature()
	if err = individual.UnmarshalBinary(p.IndividualSig); err != nil {
		return
	}
	bs := h.c.NewBitSet(len(lvl.nodes))
	var levelIndex int
	levelIndex, err = h.Partitioner.IndexAtLevel(p.Origin, int(p.Level))
	if err != nil {
		return
	}
	bs.Set(levelIndex, true)
	msind := &MultiSignature{BitSet: bs, Signature: individual}
	ind = &incomingSig{
		origin:      p.Origin,
		level:       p.Level,
		ms:          msind,
		isInd:       true,
		mappedIndex: levelIndex,
	}
	return
}
