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
		sigs  []*incomingSig
		level int
		exp   *MultiSignature
	}

	sig0 := fullIncomingSig(0)
	sig01 := *sig0
	sig01.level = 1
	sig1 := fullIncomingSig(1)
	sig2 := fullIncomingSig(2)

	sig4 := fullIncomingSig(4)
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
			store.Store(sigs)
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
	ind := &incomingSig{
		origin:      0,
		level:       0,
		ms:          &MultiSignature{BitSet: bs1, Signature: &fakeSig{true}},
		isInd:       false,
		mappedIndex: 0,
	}
	store.Store(ind )
	ms := store.FullSignature()
	require.Equal(t, n, ms.BitSet.BitLength())
	require.True(t, ms.BitSet.Get(1))
}

func TestStoreUnsafeCheckMerge(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := NewBinPartitioner(0, reg)
	store := newReplaceStore(part, NewWilffBitset, new(fakeCons))

	// We put a first sig. It should get in.
	bs1 := NewWilffBitset(4)
	bs1.Set(0, true)
	p4L3 := &incomingSig{
		origin:      1,
		level:       3,
		ms:          &MultiSignature{BitSet: bs1, Signature: &fakeSig{true}},
		isInd:       true,
		mappedIndex: 0,
	}
	s, b := store.unsafeCheckMerge(p4L3)
	require.True(t, b)
	require.True(t, s.Get(0))
	require.Equal(t, 1, s.BitSet.Cardinality())
	store.Store(p4L3)

	// If we try again we should be told that it exists already.
	s, b = store.unsafeCheckMerge(p4L3)
	require.False(t, b)
	require.Nil(t, s)

	// A larger signature should get in.
	bs1 = NewWilffBitset(4)
	bs1.Set(0, true)
	bs1.Set(2, true)
	p46L3 := &incomingSig{
		origin:      1,
		level:       3,
		ms:          &MultiSignature{BitSet: bs1, Signature: &fakeSig{true}},
		isInd:       false,
		mappedIndex: 0,
	}
	s, b = store.unsafeCheckMerge(p46L3)
	require.True(t, b)
	require.True(t, s.Get(0))
	require.True(t, s.Get(2))
	require.Equal(t, 2, s.BitSet.Cardinality())
	store.Store(p46L3)
	best, _ := store.Best(3)
	require.Equal(t, bs1.Clone(),  best.BitSet)

	// This signature is size 2 as well, but can be merged with the individual one, so
	//  we will end-up with a size 3 signature
	bs1 = NewWilffBitset(4)
	bs1.Set(3, true)
	bs1.Set(2, true)
	p67L3 := &incomingSig{
		origin:      1,
		level:       3,
		ms:          &MultiSignature{BitSet: bs1, Signature: &fakeSig{true}},
		isInd:       false,
		mappedIndex: 0,
	}
	s, b = store.unsafeCheckMerge(p67L3)
	require.True(t, b)
	require.True(t, s.Get(0))
	require.True(t, s.Get(2))
	require.True(t, s.Get(3))
	require.Equal(t, 3, s.BitSet.Cardinality())
}

func TestStoreReplace(t *testing.T) {
	n := 8
	reg := FakeRegistry(n)
	part := NewBinPartitioner(1, reg)
	sig0 := &incomingSig{level: 0, ms: fullSig(0)}
	sig1 := &incomingSig{level: 1, ms: fullSig(1)}
	sig2 := &incomingSig{level: 2, ms: fullSig(2)}
	sig3 := &incomingSig{level: 3, ms: fullSig(3)}

	fullBs3 := NewWilffBitset(n / 2)
	for i := 0; i < fullBs3.BitLength(); i++ {
		fullBs3.Set(i, true)
	}
	fullSig3 := &incomingSig{level: 3, ms: newSig(fullBs3)}
	fullBs2 := NewWilffBitset(pow2(3 - 1))
	// only signature 2 present so no 0, 1
	for i := 2; i < fullBs2.BitLength(); i++ {
		fullBs2.Set(i, true)
	}
	fullSig2 := &incomingSig{level: 3, ms: newSig(fullBs2)}

	var sc = func(ms ...int) []int {
		return ms
	}

	type storeTest struct {
		toStore []*incomingSig
		scores  []int
		ret     []bool
		best    byte
		eqMs    *MultiSignature
		eqBool  bool
		highest *incomingSig // can be nil
	}

	var s = func(sps ...*incomingSig) []*incomingSig { return sps }
	var b = func(rets ...bool) []bool { return rets }
	var tests = []storeTest{
		// empty
		{s(), sc(), b(), 2, nil, false, nil},
		// duplicate
		{s(sig2, sig2), sc(999980, 0), b(true, false), 2, sig2.ms, true, fullSig2},
		// highest
		{s(sig0, sig1, sig2, sig3), sc(1000000, 999990, 999980, 999970), b(true, true, true, true), 2, sig2.ms, true, fullSig3},
	}

	for i, test := range tests {
		t.Logf("-- test %d --", i)
		store := newReplaceStore(part, NewWilffBitset, new(fakeCons))
		for i, s := range test.toStore {
			score := store.Evaluate(s)
			require.Equal(t, test.scores[i], score)
			// then actually store the damn thing
			ret := store.Store(s)
			require.True(t, test.ret[i] == (ret != nil) )
		}
		ms, ok := store.Best(test.best)
		require.Equal(t, test.eqMs, ms)
		require.Equal(t, test.eqBool, ok)
		//require.Equal(t, test.highest, store.Highest())
	}
}
