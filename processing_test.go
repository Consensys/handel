package handel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSigProcessingStrategy(t *testing.T)  {
	n := 16
	registry := FakeRegistry(n)
	partitioner := NewBinPartitioner(1, registry)
	cons := new(fakeCons)
	sig0 := fullSigPair(0)
	sig1 := fullSigPair(1)
	sig2 := fullSigPair(2)

	ss := newSigProcessWithStrategy(partitioner, cons, nil, &EvaluatorLevel{}, nil)

	require.Equal(t, 0, len(ss.todos))
	ss.add(sig2)
	require.Equal(t, 1, len(ss.todos))

    stop := ss.processStep()
	require.Equal(t, false, stop)
	require.Equal(t, 0, len(ss.todos))

    // With the evaluator used, signatures at level 0 are discarded & signatures with
    //  an higher level are verified first.
	ss.add(sig0)
	ss.add(sig1)
	ss.add(sig2)
	ss.add(sig0)
	ss.processStep()
	require.Equal(t, 1, len(ss.todos))
	require.Equal(t, sig1, ss.todos[0])

	ss.add(&deathPillPair)
	stop2 := ss.processStep()
	require.Equal(t, true, stop2)
}

func TestProcessingFifo(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	partitioner := NewBinPartitioner(1, registry)
	cons := new(fakeCons)

	type testProcess struct {
		in  []*sigPair
		out []*sigPair
	}
	sig2 := fullSigPair(2)
	sig2Inv := fullSigPair(2)
	sig2Inv.ms.Signature.(*fakeSig).verify = false
	sig3 := fullSigPair(3)

	var s = func(sigs ...*sigPair) []*sigPair { return sigs }

	var tests = []testProcess{
		// all good, one one
		{s(sig2), s(sig2)},
		// wrong signature
		{s(sig2Inv, sig3), s(nil, sig3)},
		// The following cases test the logic of the processing, eg.
		//  skipping some validations
		// twice the same: we expect only one sig on the out chan
		//{s(sig2, sig2), s(sig2, nil)},
		// diff level:
		//{s(sig2, sig3, sig2), s(sig2, sig3, nil)},
	}

	fifo := newFifoProcessing(partitioner, cons, msg, nil).(*fifoProcessing)
	go fifo.Start()
	time.Sleep(20 * time.Millisecond)
	fifo.Stop()

	fifos := make([]signatureProcessing, 0, len(tests))
	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)

		store := newReplaceStore(partitioner, NewWilffBitset)
		fifo := newFifoProcessing(partitioner, cons, msg, nil)
		fifos = append(fifos, fifo)
		go fifo.Start()

		in := fifo.Incoming()
		require.NotNil(t, in)
		verified := fifo.Verified()
		require.NotNil(t, verified)

		// input all signature pairs
		for i, sp := range test.in {
			in <- *sp
			// expect same order of verified
			out := test.out[i]
			var s *sigPair
			select {
			case p := <-verified:
				s = &p
			case <-time.After(20 * time.Millisecond):
				s = nil
			}
			require.Equal(t, out, s)
			// simulate storage
			store.Store(sp.level, sp.ms)
		}
	}

	for _, fifo := range fifos {
		fifo.Stop()
	}
}
