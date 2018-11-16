package handel

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessingFifo(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	partitioner := newBinTreePartition(1, registry)
	cons := new(fakeCons)

	type testProcess struct {
		in  []*sigPair
		out []*verifiedSig
	}
	sig2 := fullSigPair(2)
	sig2Inv := fullSigPair(2)
	sig2Inv.ms.Signature.(*fakeSig).verify = false
	sig3 := fullSigPair(3)

	vsig2 := &verifiedSig{*sig2, true}
	vsig2f := &verifiedSig{*sig2, false}
	vsig3 := &verifiedSig{*sig3, true}

	var s = func(sigs ...*sigPair) []*sigPair { return sigs }
	var v = func(vsigs ...*verifiedSig) []*verifiedSig { return vsigs }

	var tests = []testProcess{
		// all good, one one
		{s(sig2), v(vsig2)},
		// twice the same
		{s(sig2, sig2), v(vsig2, vsig2f)},
		// diff level
		{s(sig2, sig3, sig2), v(vsig2, vsig3, vsig2f)},
		// wrong signature
		{s(sig2Inv), v()},
	}

	store := newReplaceStore(partitioner, NewWilffBitset)
	fifo := newFifoProcessing(store, partitioner, cons, msg).(*fifoProcessing)
	go fifo.Start()
	time.Sleep(20 * time.Millisecond)
	fifo.Stop()
	require.True(t, fifo.done)

	fifos := make([]signatureProcessing, 0, len(tests))
	for _, test := range tests {
		store := newReplaceStore(partitioner, NewWilffBitset)
		fifo := newFifoProcessing(store, partitioner, cons, msg)
		fifos = append(fifos, fifo)
		go fifo.Start()

		in := fifo.Incoming()
		require.NotNil(t, in)
		verified := fifo.Verified()
		require.NotNil(t, verified)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			// input all signature pairs
			for _, sp := range test.in {
				in <- *sp
			}
			wg.Done()
		}()

		// expect same order of verified
		for _, out := range test.out {
			v := <-verified
			require.Equal(t, *out, v)
		}

		wg.Wait()
	}
	for _, fifo := range fifos {
		fifo.Stop()
	}
}
