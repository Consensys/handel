package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreCombined(t *testing.T) {
	n := 9
	reg := FakeRegistry(n)

	type combineTest struct {
		id    int32
		sigs  []*sigPair
		level int
		exp   *MultiSignature
	}

	sig0 := fullSigPair(0)
	sig01 := *sig0
	sig01.level = 1
	sig1 := fullSigPair(1)
	sig2 := fullSigPair(2)

	sig4 := fullSigPair(4)
	bs5 := finalBitset(n)

	var tests = []combineTest{
		{1, sigs(sig0, sig1), 1, sig2.ms},
		{1, sigs(sig0), 0, sig01.ms},
		{8, sigs(sig0), 0, sig01.ms},
		// combined with the rest
		{8, sigs(sig0, sig4), 5, &MultiSignature{Signature: &fakeSig{true}, BitSet: bs5}},
	}

	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		part := NewBinPartitioner(test.id, reg)
		store := newReplaceStore(part, NewWilffBitset, new(fakeCons))
		for _, sigs := range test.sigs {
			store.Store(sigs.level, sigs.ms)
		}
		sp := store.Combined(byte(test.level))
		require.Equal(t, test.exp, sp)
	}
}

func TestStoreFullSignature(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := NewBinPartitioner(1, reg)
	store := newReplaceStore(part, NewWilffBitset, new(fakeCons))
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
	part := NewBinPartitioner(1, reg)
	sig0 := &sigPair{level: 0, ms: fullSig(0)}
	sig1 := &sigPair{level: 1, ms: fullSig(1)}
	sig2 := &sigPair{level: 2, ms: fullSig(2)}
	sig3 := &sigPair{level: 3, ms: fullSig(3)}

	fullBs3 := NewWilffBitset(n / 2)
	for i := 0; i < fullBs3.BitLength(); i++ {
		fullBs3.Set(i, true)
	}
	fullSig3 := &sigPair{level: 3, ms: newSig(fullBs3)}
	fullBs2 := NewWilffBitset(pow2(3 - 1))
	// only signature 2 present so no 0, 1
	for i := 2; i < fullBs2.BitLength(); i++ {
		fullBs2.Set(i, true)
	}
	fullSig2 := &sigPair{level: 3, ms: newSig(fullBs2)}

	var sc = func(ms ...int) []int {
		return ms
	}

	type storeTest struct {
		toStore []*sigPair
		scores  []int
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
		{s(), sc(), b(), 2, nil, false, nil},
		// duplicate
		{s(sig2, sig2), sc(1, 0), b(true, false), 2, sig2.ms, true, fullSig2},
		// highest
		{s(sig0, sig1, sig2, sig3), sc(1, 1, 1, 1), b(true, true, true, true), 2, sig2.ms, true, fullSig3},
	}

	for i, test := range tests {
		t.Logf("-- test %d --", i)
		store := newReplaceStore(part, NewWilffBitset, new(fakeCons))
		for i, s := range test.toStore {
			score := store.Evaluate(s)
			require.Equal(t, test.scores[i], score)
			// then actually store the damn thing
			_, ret := store.Store(s.level, s.ms)
			require.Equal(t, test.ret[i], ret)
		}
		ms, ok := store.Best(test.best)
		require.Equal(t, test.eqMs, ms)
		require.Equal(t, test.eqBool, ok)
		//require.Equal(t, test.highest, store.Highest())
	}
}
