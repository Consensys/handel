package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreHighest(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := newBinTreePartition(1, reg)
	store := newReplaceStore(part, NewWilffBitset)

	store.Store(3, &MultiSignature{BitSet: NewWilffBitset(4), Signature: &fakeSig{true}})
	store.Store(2, &MultiSignature{BitSet: NewWilffBitset(2), Signature: &fakeSig{true}})
	pair := store.Highest()

	require.NotNil(t, pair)
	require.Equal(t, 4, int(pair.level))
	require.Equal(t, 8, pair.ms.BitSet.BitLength())

	// weird case with 0 and 1
	store = newReplaceStore(part, NewWilffBitset)
	store.Store(0, &MultiSignature{BitSet: NewWilffBitset(1), Signature: &fakeSig{true}})
	store.Store(1, &MultiSignature{BitSet: NewWilffBitset(1), Signature: &fakeSig{true}})
	pair = store.Highest()

}

func TestStoreFullSignature(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := newBinTreePartition(1, reg)
	store := newReplaceStore(part, NewWilffBitset)
	bs1 := NewWilffBitset(1)
	bs1.Set(0, true)

	store.Store(0, &MultiSignature{BitSet: bs1, Signature: &fakeSig{true}})
	ms := store.FullSignature()
	require.Equal(t, n, ms.BitSet.BitLength())
	require.True(t, ms.BitSet.Get(1))
}

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
	fullSig3 := &sigPair{4, newSig(fullBs3)}
	fullBs2 := NewWilffBitset(pow2(3 - 1))
	// only signature 2 present so no 0, 1
	for i := 2; i < fullBs2.BitLength(); i++ {
		fullBs2.Set(i, true)
	}
	fullSig2 := &sigPair{level: 3, ms: newSig(fullBs2)}

	// preparing mocked type return
	type mockRet struct {
		ms    *MultiSignature
		isNew bool
	}
	var mr = func(ms *MultiSignature, isNew bool) *mockRet {
		return &mockRet{ms: ms, isNew: isNew}
	}
	var m = func(ms ...*mockRet) []*mockRet {
		return ms
	}
	sig0Ret := mr(sig0.ms, true)
	sig1Ret := mr(sig1.ms, true)
	sig2Ret := mr(sig2.ms, true)
	sig2Retf := mr(sig2.ms, false)
	sig3Ret := mr(sig3.ms, true)

	type storeTest struct {
		toStore  []*sigPair
		mockRets []*mockRet
		ret      []bool
		best     byte
		eqMs     *MultiSignature
		eqBool   bool
		highest  *sigPair // can be nil
	}

	var s = func(sps ...*sigPair) []*sigPair { return sps }
	var b = func(rets ...bool) []bool { return rets }
	var tests = []storeTest{
		// empty
		{s(), m(mr(nil, false)), b(), 2, nil, false, nil},
		// duplicate
		{s(sig2, sig2), m(sig2Ret, sig2Retf), b(true, false), 2, sig2.ms, true, fullSig2},
		// highest
		{s(sig0, sig1, sig2, sig3), m(sig0Ret, sig1Ret, sig2Ret, sig3Ret), b(true, true, true, true), 2, sig2.ms, true, fullSig3},
	}

	for i, test := range tests {
		t.Logf("-- test %d --", i)
		store := newReplaceStore(part, NewWilffBitset)
		for i, s := range test.toStore {
			// first mimick the storing and test if returns fits
			newSig, isNew := store.MockStore(s.level, s.ms)
			require.Equal(t, test.mockRets[i].ms, newSig)
			require.Equal(t, test.mockRets[i].isNew, isNew)
			// then actually store the damn thing
			_, ret := store.Store(s.level, s.ms)
			require.Equal(t, test.ret[i], ret)
		}
		ms, ok := store.Best(test.best)
		require.Equal(t, test.eqMs, ms)
		require.Equal(t, test.eqBool, ok)
		require.Equal(t, test.highest, store.Highest())
	}
}
