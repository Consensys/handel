package handel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type EvaluatorLevel struct {
}

func (f *EvaluatorLevel) Evaluate(sp *incomingSig) int {
	return int(sp.level)
}

func TestSigProcessingStrategy(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	partitioner := NewBinPartitioner(1, registry, DefaultLogger)
	cons := new(fakeCons)
	sig0 := fullIncomingSig(0)
	sig1 := fullIncomingSig(1)
	sig2 := fullIncomingSig(2)

	s := newEvaluatorProcessing(nil, partitioner, cons, nil, 0, &EvaluatorLevel{}, nil)
	ss := s.(*evaluatorProcessing)

	require.Equal(t, 0, len(ss.todos))
	ss.Add(sig2)
	require.Equal(t, 1, len(ss.todos))

	stop := ss.processStep()
	require.Equal(t, false, stop)
	require.Equal(t, 0, len(ss.todos))

	// With the evaluator used, signatures at level 0 are discarded & signatures with
	//  an higher level are verified first.
	ss.Add(sig0)
	ss.Add(sig1)
	ss.Add(sig2)
	ss.Add(sig0)
	ss.processStep()
	require.Equal(t, 1, len(ss.todos))
	require.Equal(t, sig1, ss.todos[0])

	ss.Add(&deathPillPair)
	stop2 := ss.processStep()
	require.Equal(t, true, stop2)
}

func TestProcessingFifo(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	partitioner := NewBinPartitioner(1, registry, DefaultLogger)
	cons := new(fakeCons)
	store := newReplaceStore(partitioner, NewWilffBitset, cons)

	type testProcess struct {
		in  []*incomingSig
		out []*incomingSig
	}
	sig2 := fullIncomingSig(2)
	sig2Inv := fullIncomingSig(2)
	sig2Inv.ms.Signature.(*fakeSig).verify = false
	sig3 := fullIncomingSig(3)

	var s = func(sigs ...*incomingSig) []*incomingSig { return sigs }

	var tests = []testProcess{
		// all good, one one
		{s(sig2), s(sig2)},
		// wrong signature
		{s(sig2Inv, sig3), s(nil, sig3)},
		// The following cases test the logic of the processing, eg.
		//  skipping some validations
		// twice the same: we expect only one sig on the out chan
		{s(sig2, sig2), s(sig2, nil)},
		// diff level:
		{s(sig2, sig3, sig2), s(sig2, sig3, nil)},
	}

	fifo := newFifoProcessing(store, partitioner, cons, msg).(*fifoProcessing)
	go fifo.Start()
	time.Sleep(20 * time.Millisecond)
	fifo.Stop()

	fifos := make([]signatureProcessing, 0, len(tests))
	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)

		store := newReplaceStore(partitioner, NewWilffBitset, cons)
		fifo := newFifoProcessing(store, partitioner, cons, msg)
		fifos = append(fifos, fifo)
		go fifo.Start()

		verified := fifo.Verified()
		require.NotNil(t, verified)

		// input all signature pairs
		for i, sp := range test.in {
			fifo.Add(sp)
			// expect same order of verified
			out := test.out[i]
			var s *incomingSig
			select {
			case p := <-verified:
				s = &p
			case <-time.After(20 * time.Millisecond):
				s = nil
			}
			require.Equal(t, out, s)
			// simulate storage
			store.Store(sp)
		}
	}

	for _, fifo := range fifos {
		fifo.Stop()
	}
}

type fakeBlackList struct {
	byzPeers []int32
}

func (f *fakeBlackList) Update(id int32, err error) {
	f.byzPeers = append(f.byzPeers, id)
}

func (f *fakeBlackList) IsBlackListed(id int32) bool {
	return contains(f.byzPeers, id)
}

func TestVerifyAndPublish(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	partitioner := NewBinPartitioner(1, registry, DefaultLogger)
	cons := new(fakeCons)
	bl := &fakeBlackList{}
	s := newEvaluatorProcessing(bl, partitioner, cons, nil, 0, &EvaluatorLevel{}, nil)
	ss := s.(*evaluatorProcessing)
	sig1 := incorrectIncomingSig(2)
	sig1.id = 1
	sig2 := incorrectIncomingSig(2)
	sig2.id = 2
	sig3 := incorrectIncomingSig(2)
	sig3.id = 3
	sig4 := fullIncomingSig(2)
	sig4.id = 4

	ss.verifyAndPublish(sig1)
	ss.verifyAndPublish(sig2)
	ss.verifyAndPublish(sig3)
	ss.verifyAndPublish(sig4)

	require.True(t, bl.IsBlackListed(1))
	require.True(t, bl.IsBlackListed(2))
	require.True(t, bl.IsBlackListed(3))
	require.False(t, bl.IsBlackListed(4))
}

func contains(s []int32, e int32) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
