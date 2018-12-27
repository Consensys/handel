package handel

// this contains the logic for processing signatures asynchronously. Each
// incoming packet from the network is passed down to the signatureProcessing
// interface, and may be returned to main Handel logic when verified.

import (
	"errors"
	"fmt"
	. "math"
	"sync"
)

var deathPillPair = sigPair{-1, 121, nil}

// signatureProcessing is an interface responsible for verifying incoming
// multi-signature. It can decides to drop some incoming signatures if deemed
// useless. It outputs verified signatures to the main handel processing logic
// It is an asynchronous processing interface that needs to be started and
// stopped when needed.
type signatureProcessing interface {
	// Start is a blocking call that starts the processing routine
	Start()
	// Stop is a blocking call that stops the processing routine
	Stop()
	// channel upon which to send new incoming signatures
	Incoming() chan sigPair
	// channel that outputs verified signatures. Implementation must guarantee
	// that all verified signatures are signatures that have been sent on the
	// incoming channel. No new signatures must be outputted on this channel (
	// is the role of the Store)
	Verified() chan sigPair
}

type sigProcessWithStrategy struct {
	c *sync.Cond

	part Partitioner
	cons Constructor
	msg  []byte

	out           chan sigPair
	lastCompleted int
	todos         []sigPair
	dones         map[int]int
	bests         map[int]*sigPair
}

func newSigProcessWithStrategy(part Partitioner, c Constructor, msg []byte) *sigProcessWithStrategy {
	m := sync.Mutex{}
	return &sigProcessWithStrategy{
		c:    sync.NewCond(&m),
		part: part,
		cons: c,
		msg:  msg,

		out:           make(chan sigPair, 1000),
		lastCompleted: 0,
		todos:         make([]sigPair, 0),
		dones:         make(map[int]int),
		bests:         make(map[int]*sigPair),
	}
}

// fifoProcessing implements the signatureProcessing interface using a simple
// fifo queue, verifying all incoming signatures, not matter relevant or not.
type fifoProcessing struct {
	in   chan sigPair
	proc *sigProcessWithStrategy
}

func (f *sigProcessWithStrategy) add(sp sigPair) {
	f.c.L.Lock()
	defer f.c.L.Unlock()

	if int(sp.level) > f.lastCompleted {
		f.todos = append(f.todos, sp)
		f.c.Broadcast()
	}
}

// Look at the signatures received so far and select the ones
//  that should be processed first.
func (f *sigProcessWithStrategy) readTodos() bool {
	f.c.L.Lock()
	defer f.c.L.Unlock()
	for ; len(f.todos) == 0; {
		f.c.Wait()
	}

	for _, pair := range f.todos {
		if pair == deathPillPair {
			return true
		}
		if pair.ms == nil {
			continue
		}
		lvl := int(pair.level)
		if lvl > f.lastCompleted {
			card := pair.ms.Cardinality()
			if f.dones[lvl] < card && (f.bests[lvl] == nil || f.bests[lvl].ms.Cardinality() < card) {
				f.bests[lvl] = &pair
				break //todo -- we don't want to break here
			}
		}
	}
	f.todos = make([]sigPair, 0)
	return false
}

func (f *sigProcessWithStrategy) hasTodos() bool {
	f.c.L.Lock()
	defer f.c.L.Unlock()
	return len(f.todos) > 0
}

func (f *sigProcessWithStrategy) process() {
	sigCount := 0
	for stop := false; !stop; stop = f.readTodos() {
		if  !f.hasTodos() && len(f.bests) > 0 {
			minLevel := MaxInt32
			for lvl := range f.bests {
				if minLevel > lvl {
					minLevel = lvl
					break
				}
			}
			choice := f.bests[minLevel]
			delete(f.bests, minLevel)

			err := f.verifySignature(choice)
			if err != nil {
				// todo: we have a bug here as we may have eliminated possibly valid signatures
				logf("handel: fifo: verifying err: %s", err)
			} else {
				f.dones[minLevel] = choice.ms.Cardinality()
				f.out <- *choice
				sigCount++
				if sigCount % 100 == 0 {
					logf("Processed %d signatures", sigCount)
				}
			}
		}
	}

	close(f.out)
}

//****************************
// We should:
//   - skip the signatures we receive for completed levels (require an info from handel)
//   - check if we can complete a signature we have already verified (this is important for censorship resistance)
//
// Possible strategy:
//   - look at all signatures in the pipeline
//   - sort them by level (be careful
//****************

// newFifoProcessing returns a signatureProcessing implementation using a fifo
// queue. It needs the store to store the valid signatures, the partitioner +
// constructor + msg to verify the signatures.
func newFifoProcessing(part Partitioner, c Constructor, msg []byte) signatureProcessing {
	proc := newSigProcessWithStrategy(part, c, msg)
	go proc.process()

	return &fifoProcessing{
		in:   make(chan sigPair, 1000),
		proc: proc,
	}
}

// processIncoming simply verifies the signature, stores it, and outputs it
func (f *fifoProcessing) processIncoming() {
	for pair := range f.in {
		f.proc.add(pair)
		if pair == deathPillPair {
			f.close()
			return
		}
	}
}

func (f *sigProcessWithStrategy) verifySignature(pair *sigPair) error {
	level := pair.level
	if level <= 0 {
		panic("level <= 0")
	}
	ms := pair.ms
	ids, err := f.part.IdentitiesAt(int(level))
	if err != nil {
		return err
	}

	if ms.BitSet.BitLength() != len(ids) {
		return errors.New("handel: inconsistent bitset with given level")
	}

	// compute the aggregate public key corresponding to bitset
	aggregateKey := f.cons.PublicKey()
	for i := 0; i < ms.BitSet.BitLength(); i++ {
		if !ms.BitSet.Get(i) {
			continue
		}
		aggregateKey = aggregateKey.Combine(ids[i].PublicKey())
	}

	if err := aggregateKey.VerifySignature(f.msg, ms.Signature); err != nil {
		logf("processing err: from %d -> level %d -> %s", pair.origin, pair.level, ms.String())
		return fmt.Errorf("handel: %s", err)
	}
	return nil
}

func (f *fifoProcessing) Incoming() chan sigPair {
	return f.in
}

func (f *fifoProcessing) Verified() chan sigPair {
	return f.proc.out
}

func (f *fifoProcessing) Start() {
	f.processIncoming()
}

func (f *fifoProcessing) Stop() {
	f.in <- deathPillPair
}

func (f *fifoProcessing) close() {
	close(f.in)
}
