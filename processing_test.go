package handel

import (
	"fmt"
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
		fmt.Printf(" --++++++-- test %d -- \n", i)

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
			var s *sigPair
			select {
			case p := <-verified:
				s = &p
			case <-time.After(20 * time.Millisecond):
				s = nil
			}
			fmt.Println("test.out = ", out, " vs fetched ", s)
			require.Equal(t, out, s)
		}

		wg.Wait()
	}
	for _, fifo := range fifos {
		fifo.Stop()
	}
}
