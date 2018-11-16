package handel

// this contains the logic for processing signatures asynchronously. Each
// incoming packet from the network is passed down to the signatureProcessing
// interface, and may be returned to main Handel logic when verified.

import (
	"errors"
	"fmt"
	"sync"
)

// signatureProcessing is an interface responsible for processing incoming
// multi-signature: verifying them, if needed, and storing them, if needed. It
// outputs verified signatures to the main handel processing logic It is an
// asynchronous processing interface that needs to be started and stopped when
// needed.
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
	Verified() chan verifiedSig
}

type verifiedSig struct {
	sigPair
	new bool
}

// fifoProcessing implements the signatureProcessing interface using a simple
// fifo queue, verifying all incoming signatures, not matter relevant or not.
type fifoProcessing struct {
	sync.Mutex
	store signatureStore
	part  partitioner
	cons  Constructor
	msg   []byte
	in    chan sigPair
	out   chan verifiedSig
	done  bool
}

// newFifoProcessing returns a signatureProcessing implementation using a fifo
// queue. It needs the store to store the valid signatures, the partitioner +
// constructor + msg to verify the signatures.
func newFifoProcessing(store signatureStore, part partitioner,
	c Constructor, msg []byte) signatureProcessing {
	return &fifoProcessing{
		part:  part,
		store: store,
		cons:  c,
		msg:   msg,
		in:    make(chan sigPair, 100),
		out:   make(chan verifiedSig, 100),
	}
}

// processIncoming simply verifies the signature, stores it, and outputs it
func (f *fifoProcessing) processIncoming() {
	for pair := range f.in {
		err := f.verifySignature(&pair)
		if err != nil {
			logf(err.Error())
		}
		new := f.store.Store(pair.level, pair.ms)
		logf("handel: processing verified and stored one new signature")
		if f.isStopped() {
			break
		}
		f.out <- verifiedSig{pair, new}
	}
}

func (f *fifoProcessing) verifySignature(pair *sigPair) error {
	level := pair.level
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
		return fmt.Errorf("handel: %s", err)
	}
	return nil
}

func (f *fifoProcessing) Incoming() chan sigPair {
	return f.in
}

func (f *fifoProcessing) Verified() chan verifiedSig {
	return f.out
}

func (f *fifoProcessing) Start() {
	f.processIncoming()
}

func (f *fifoProcessing) Stop() {
	f.Lock()
	defer f.Unlock()
	if f.done {
		return
	}
	f.done = true
	close(f.in)
	close(f.out)
}

func (f *fifoProcessing) isStopped() bool {
	f.Lock()
	defer f.Unlock()
	// OK since once we call stop, we'll no go back to done = false
	return f.done
}
