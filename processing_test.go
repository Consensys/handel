package handel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
		// twice the same
		{s(sig2, sig2), s(sig2, nil)},
		// diff level
		{s(sig2, sig3, sig2), s(sig2, sig3, nil)},
		// wrong signature
		{s(sig2Inv), s(nil)},
	}

	store := newReplaceStore(partitioner, NewWilffBitset)
	fifo := newFifoProcessing(store, partitioner, cons, msg).(*fifoProcessing)
	go fifo.Start()
	time.Sleep(20 * time.Millisecond)
	fifo.Stop()
	require.True(t, fifo.done)

	fifos := make([]signatureProcessing, 0, len(tests))
	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)

		store := newReplaceStore(partitioner, NewWilffBitset)
		fifo := newFifoProcessing(store, partitioner, cons, msg)
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
