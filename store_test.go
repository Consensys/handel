package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreReplace(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := newBinTreePartition(1, reg)
	sig0 := &sigPair{0, fullSig(0)}
	sig1 := &sigPair{1, fullSig(1)}
	sig2 := &sigPair{2, fullSig(2)}
	sig3 := &sigPair{3, fullSig(3)}

	fullBs3 := NewWilffBitset(n)
	for i := 0; i < n; i++ {
		fullBs3.Set(i, true)
	}
	fullSig3 := &sigPair{3, newSig(fullBs3)}
	fullBs2 := NewWilffBitset(n)
	// only signature 2 present so no 0, 1
	for i := 2; i < n/2; i++ {
		fullBs2.Set(i, true)
	}
	fullSig2 := &sigPair{2, newSig(fullBs2)}

	type storeTest struct {
		toStore []*sigPair
		ret     []bool
		best    byte
		eqMs    *MultiSignature
		eqBool  bool
		highest *sigPair // can be nil
	}

	var s = func(sps ...*sigPair) []*sigPair { return sps }
	var b = func(rets ...bool) []bool { return rets }

	var tests = []storeTest{
		// empty
		{s(), b(), 2, nil, false, nil},
		// duplicate
		{s(sig2, sig2), b(true, false), 2, sig2.ms, true, fullSig2},
		// highest
		{s(sig0, sig1, sig2, sig3), b(true, true, true, true), 2, sig2.ms, true, fullSig3},
	}

	for i, test := range tests {
		t.Logf("-- test %d --", i)
		store := newReplaceStore(part, NewWilffBitset)
		for i, s := range test.toStore {
			ret := store.Store(s.level, s.ms)
			require.Equal(t, test.ret[i], ret)
		}
		ms, ok := store.Best(test.best)
		require.Equal(t, test.eqMs, ms)
		require.Equal(t, test.eqBool, ok)
		require.Equal(t, test.highest, store.BestCombined())
	}
}
