package handel

// this contains the logic for processing signatures asynchronously. Each
// incoming packet from the network is passed down to the signatureProcessing
// interface, and may be returned to main Handel logic when verified.

import (
	"errors"
	"fmt"
	"sync"
)

var deathPillPair = sigPair{-1, 121, nil}

// signatureProcessing is an interface responsible for verifying incoming
// multi-signature. It can decides to drop some incoming signatures if deemed
// useless. It outputs verified signatures to the main handel processing logic
// It is an asynchronous processing interface that needs to be sendStarted and
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

type SigEvaluator interface {
	// Evaluate the interest to verify a signature
	//   0: no interest, the signature can be discarded definitively
	//  >0: the greater the more interesting
	Evaluate(sp *sigPair) int
}

type Evaluator1 struct {
}

func (f *Evaluator1) Evaluate(sp *sigPair) int {
	return 1
}

func newEvaluator1() SigEvaluator {
	return &Evaluator1{}
}

type EvaluatorStore struct {
	store signatureStore
}

func (f *EvaluatorStore) Evaluate(sp *sigPair) int {
	ms, ok := f.store.Best(sp.level)
	if ok && ms.Cardinality() >= sp.ms.Cardinality() {
		//return 0
	}
	return 1
}

func newEvaluatorStore(store signatureStore) SigEvaluator {
	return &EvaluatorStore{store:store}
}

type sigProcessWithStrategy struct {
	cond *sync.Cond

	h *Handel

	part Partitioner
	cons Constructor
	msg  []byte

	out       chan sigPair
	todos     []*sigPair
	evaluator SigEvaluator
}

func newSigProcessWithStrategy(part Partitioner, c Constructor, msg []byte, e SigEvaluator, h *Handel) *sigProcessWithStrategy {
	m := sync.Mutex{}

	return &sigProcessWithStrategy{
		cond: sync.NewCond(&m),
		part: part,
		cons: c,
		msg:  msg,

		out:       make(chan sigPair, 1000),
		todos:     make([]*sigPair, 0),
		evaluator: e,
		h : h,
	}
}

// fifoProcessing implements the signatureProcessing interface using a simple
// fifo queue, verifying all incoming signatures, not matter relevant or not.
type fifoProcessing struct {
	in   chan sigPair
	proc *sigProcessWithStrategy
}

func (f *sigProcessWithStrategy) add(sp *sigPair) {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()

	f.todos = append(f.todos, sp)
	if f.h != nil && false {
		f.h.logf("added %s", sp)
	}
	f.cond.Signal()
}

// Look at the signatures received so far and select the one
//  that should be processed first.
func (f *sigProcessWithStrategy) readTodos() (bool, *sigPair) {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	for len(f.todos) == 0 {
		if f.h != nil  && false{
			f.h.logf("waiting, todos is empty")
		}
		f.cond.Wait()
	}
	if f.h != nil && false {
		f.h.logf("readTodos %v", f.todos)
	}

	// We need to iterate on our list. We put in
	//   'newTodos' the signatures not selected in this round
	//   but possibly interesting next time
	newTodos := make([]*sigPair, 0)
	var best *sigPair
	bestMark := 0
	for _, pair := range f.todos {
		if *pair == deathPillPair {
			return true, nil
		}
		if pair.ms == nil {
			continue
		}

		mark := f.evaluator.Evaluate(pair)
		if mark > 0 {
			if mark <= bestMark {
				newTodos = append(newTodos, pair)
			} else {
				if best != nil {
					newTodos = append(newTodos, best)
				}
				best = pair
				bestMark = mark
			}
		}
	}

	f.todos = newTodos
	return false, best
}

func (f *sigProcessWithStrategy) hasTodos() bool {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	return len(f.todos) > 0
}

func (f *sigProcessWithStrategy) processLoop() {
	sigCount := 0
	for {
		stop := f.processStep()
		if stop {
			return
		}
		sigCount++
		if sigCount%100 == 0 {
			logf("Processed %d signatures", sigCount)
		}
	}
}

func (f *sigProcessWithStrategy) processStep() (bool) {
	done, best := f.readTodos()
	if done {
		close(f.out)
		return true
	}
	if best != nil {
		f.verifyAndPublish(best)
	}
	return false
}

func (f *sigProcessWithStrategy) verifyAndPublish(sp *sigPair) {
	err := f.verifySignature(sp)
	if err != nil {
		logf("fifo: verifying err: %s", err)
	} else {
		f.out <- *sp
	}
}

// newFifoProcessing returns a signatureProcessing implementation using a fifo
// queue. It needs the store to store the valid signatures, the partitioner +
// constructor + msg to verify the signatures.
func newFifoProcessing(part Partitioner, c Constructor, msg []byte, h *Handel) signatureProcessing {
	proc := newSigProcessWithStrategy(part, c, msg, newEvaluator1(), h)
	go proc.processLoop()

	return &fifoProcessing{
		in:   make(chan sigPair, 1000),
		proc: proc,
	}
}

// processIncoming simply verifies the signature, stores it, and outputs it
func (f *fifoProcessing) processIncoming() {
	async := true
	for pair := range f.in {
		if async {
			p := pair
			f.proc.add(&p)
		}
		if pair == deathPillPair {
			f.close()
			return
		} else {
			if !async {
				f.proc.verifyAndPublish(&pair)
			}
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

