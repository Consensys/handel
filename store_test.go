package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreReplace(t *testing.T) {
	sig2 := &sigPair{2, fullSig(2)}
	sig3 := &sigPair{3, fullSig(3)}

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
		{s(sig2, sig2), b(true, false), 2, sig2.ms, true, sig2},
		// highest
		{s(sig2, sig3), b(true, true), 2, sig2.ms, true, sig3},
	}

	for _, test := range tests {
		store := newReplaceStore()
		for i, s := range test.toStore {
			ret := store.Store(s.level, s.ms)
			require.Equal(t, test.ret[i], ret)
		}
		ms, ok := store.Best(test.best)
		require.Equal(t, test.eqMs, ms)
		require.Equal(t, test.eqBool, ok)
		require.Equal(t, test.highest, store.Highest())
	}
}
