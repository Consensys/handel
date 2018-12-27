package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionerBinTreeSize(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	type sizeTest struct {
		id    int32
		level int
		isErr bool
		exp   int
	}

	var tests = []sizeTest{
		{1, 0, false, 1},
		{1, 1, false, 1},
		{1, 2, false, 2},
		{1, 3, false, 4},
		{1, 4, false, 8},
		// 1 because 17 is alone in his group (only node > 16 )
		{1, 5, false, 1},
		{1, 6, false, 17},
		// -- 17's point of view
		{16, 0, false, 1},
		{16, 5, false, n - 1},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		ct := NewBinPartitioner(test.id, reg)
		size := ct.Size(test.level)
		require.Equal(t, test.exp, size)
	}
}

func TestPartitionerBinTreeCombine(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	type combineTest struct {
		id    int32
		sigs  []*sigPair
		level int
		isErr bool
		exp   *MultiSignature
	}
	sig3 := &fakeSig{true}
	bs3 := NewWilffBitset(n / 2)
	for i := 0; i < bs3.BitLength(); i++ {
		bs3.Set(i, true)
	}
	sig2 := &fakeSig{true}
	bs2 := NewWilffBitset(pow2(3 - 1))
	// only the level-2 bits are set
	for i := 2; i < 4; i++ {
		bs2.Set(i, true)
	}

	bs0 := NewWilffBitset(1)
	bs0.Set(0, true)

	// final signature should have level 4 and bitlength 8
	// with first bit set to false
	pairs3 := sigPairs(0, 2, 3)
	final4 := finalSigPair(4, 8)
	final4.ms.BitSet.Set(0, false)

	fullBs := NewWilffBitset(n)
	for i := 0; i < n; i++ {
		fullBs.Set(i, true)
	}

	var tests = []combineTest{
		// from last node
		{16, sigPairs(0), 1, false, &MultiSignature{Signature: sig3, BitSet: bs0}},
		// error in the level requested
		{16, sigPairs(0, 5), 3, true, &MultiSignature{Signature: sig3, BitSet: fullBs}},
		// contributions from last node + all previous nodes
		{16, sigPairs(0, 5), 6, false, &MultiSignature{Signature: sig3, BitSet: fullBs}},
		// all good, we should have the first half of signature returned (
		{1, sigPairs(0, 1, 2, 3), 4, false, &MultiSignature{Signature: sig3, BitSet: bs3}},
		// only one to combine
		{1, sigPairs(2), 3, false, &MultiSignature{Signature: sig2, BitSet: bs2}},
		{1, nil, 0, true, nil},
		// with holes
		{1, pairs3, 4, false, final4.ms},
	}

	for i, test := range tests {
		ct := NewBinPartitioner(test.id, reg)
		t.Logf(" -- test %d -- ", i)
		ms := ct.Combine(test.sigs, test.level, NewWilffBitset)
		if ms == nil {
			if test.isErr {
				continue
			}
			require.NotNil(t, ms)
		}
		require.Equal(t, test.exp.Signature, ms.Signature)
		require.Equal(t, test.exp.BitSet.BitLength(), ms.BitSet.BitLength())

		bs1 := test.exp.BitSet
		bs2 := ms.BitSet
		for i := 0; i < bs1.BitLength(); i++ {
			require.Equal(t, bs1.Get(i), bs2.Get(i))
		}
	}
}

func TestPartitionerBinTreeCombineFull(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	sig3 := &fakeSig{true}
	bs3 := NewWilffBitset(n)
	for i := 0; i < pow2(3); i++ {
		bs3.Set(i, true)
	}
	sig2 := &fakeSig{true}
	bs2 := NewWilffBitset(n)
	// only the second sig is there so no 0,1
	for i := 2; i < pow2(2); i++ {
		bs2.Set(i, true)
	}

	pairs3 := sigPairs(0, 1, 2, 3, 4)
	// 4-1 -> because that's how you compute the size of a level
	// -1 -> to just spread out holes to other levels and leave this one still
	// having one contribution
	for i := 0; i < pow2(4-1)-1; i++ {
		pairs3[4].ms.BitSet.Set(i, false)
	}
	pairs3[3].ms.BitSet.Set(1, false)
	pairs3[3].ms.BitSet.Set(2, false)

	final4 := finalSigPair(4, n)
	final4.ms.BitSet.Set(5, false)
	final4.ms.BitSet.Set(6, false)
	for i := 8; i < 15; i++ {
		final4.ms.BitSet.Set(i, false)
	}
	final4.ms.BitSet.Set(16, false) // no signature of the last node included

	bs := func(is ...int) BitSet {
		fullBs := NewWilffBitset(n)
		if len(is) == 0 {
			for i := 0; i < n; i++ {
				fullBs.Set(i, true)
			}
			return fullBs
		}
		for _, i := range is {
			fullBs.Set(i, true)
		}
		return fullBs
	}

	type combineTest struct {
		id    int32
		sigs  []*sigPair
		isErr bool
		exp   *MultiSignature
	}

	var tests = []combineTest{
		// from last node
		{16, sigPairs(0), false, &MultiSignature{Signature: sig3, BitSet: bs(16)}},
		// error in the level requested
		{16, sigPairs(0, 5), true, &MultiSignature{Signature: sig3, BitSet: bs()}},
		// contributions from last node + all previous nodes
		{16, sigPairs(0, 5), false, &MultiSignature{Signature: sig3, BitSet: bs()}},

		// all good, we should have the first half of signature returned (
		{1, sigPairs(0, 1, 2, 3), false, &MultiSignature{Signature: sig3, BitSet: bs3}},
		// only one to combine
		{1, sigPairs(2), false, &MultiSignature{Signature: sig2, BitSet: bs2}},
		{1, nil, true, nil},
		// with holes
		{1, pairs3, false, final4.ms},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		ct := NewBinPartitioner(test.id, reg)
		res := ct.CombineFull(test.sigs, NewWilffBitset)
		if res == nil {
			if test.exp == nil {
				continue
			}
			t.Fatal("should not have got nil output")
		}
		require.Equal(t, test.exp.Signature, res.Signature)
		require.Equal(t, test.exp.BitSet.BitLength(), res.BitSet.BitLength())
		require.Equal(t, n, res.BitSet.BitLength())

		bs1 := test.exp.BitSet
		bs2 := res.BitSet
		for i := 0; i < bs1.BitLength(); i++ {
			require.Equal(t, bs1.Get(i), bs2.Get(i))
		}
	}
}

func TestPartitionerBinTreeMaxLevel(t *testing.T) {
	type maxLevelTest struct {
		n   int
		exp int
	}

	var tests = []maxLevelTest{
		{8, 3}, {16, 4}, {2, 1},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		reg := FakeRegistry(test.n)
		ct := NewBinPartitioner(1, reg)
		require.Equal(t, test.exp, ct.MaxLevel())
	}
}

func TestPartitionerBinTreePickNextAt(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)
	ct := NewBinPartitioner(1, reg)

	type pickTest struct {
		level int
		// how many to pick each time
		count         int
		expectedLens  []int
		expectedBools []bool
	}

	tests := []pickTest{
		// all good
		{1, 1, []int{1, 0}, []bool{true, false}},
		// larger count than available
		{2, 10, []int{2, 0}, []bool{true, false}},
		// multiple times
		{3, 2, []int{2, 2, 0}, []bool{true, true, false}},
	}

	for _, test := range tests {
		for i, lenght := range test.expectedLens {
			ids, b := ct.PickNextAt(test.level, test.count)
			require.Equal(t, lenght, len(ids))
			require.Equal(t, test.expectedBools[i], b)
		}

	}
}

func TestPartitionerBinTreeRangeAt(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	type rangeTest struct {
		id    int32
		level int
		isErr bool
		from  int
		to    int
	}

	tests := []rangeTest{
		{1, 0, false, 1, 2},
		{1, 1, false, 0, 1},
		{1, 2, false, 2, 4},
		{1, 3, false, 4, 8},
		{1, 4, false, 8, 16},
		{1, 5, false, 16, 17},
		{16, 0, false, 16, 17},
		{16, 1, true, 16, 17},
		{16, 2, true, 16, 17},
		{16, 3, true, 16, 17},
		{16, 4, true, 16, 17},
		{16, 5, false, 0, 16},
		{1, 7, true, 0, 0},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		ct := NewBinPartitioner(test.id, reg).(*binomialPartitioner)
		_ids, err := ct.IdentitiesAt(test.level)
		if test.isErr {
			require.Error(t, err)
		}
		min, max, err := ct.rangeLevel(test.level)
		if test.isErr {
			require.Error(t, err)
			continue
		}
		require.Equal(t, test.from, min)
		require.Equal(t, test.to, max)

		expected, ok := reg.Identities(test.from, test.to)
		require.True(t, ok)
		require.Equal(t, expected, _ids)
	}
}

func TestPartitionerBinTreeRangeAtInverse(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	type rangeTest struct {
		id    int32
		level int
		isErr bool
		from  int
		to    int
	}

	tests := []rangeTest{
		// test for id in the lower part of the ID space => complete power of
		// two
		{1, 0, false, 1, 2},
		{1, 1, false, 1, 2},
		{1, 2, false, 0, 2},
		{1, 3, false, 0, 4},
		{1, 4, false, 0, 8},
		{1, 5, false, 0, 16},
		//special high level where we take the whole ID space
		{1, 6, false, 0, 17},
		{1, 7, true, 0, 0},

		// test after the power of two - 16 - so the levels should be all equal
		// size = 1
		{16, 0, false, 16, 17},
		{16, 1, false, 16, 17},
		{16, 2, false, 16, 17},
		{16, 3, false, 16, 17},
		{16, 4, false, 16, 17},
		{16, 5, false, 16, 17},
		// special high level where we take the whole ID space
		{16, 6, false, 0, 17},
		{16, 7, true, 16, 17},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		ct := NewBinPartitioner(test.id, reg).(*binomialPartitioner)
		min, max, err := ct.rangeLevelInverse(test.level)
		if test.isErr {
			require.Error(t, err)
			continue
		}
		require.Equal(t, test.from, min)
		require.Equal(t, test.to, max)
	}
}

func TestIsSet(t *testing.T) {
	type setTest struct {
		nb       uint
		idx      uint
		expected bool
	}

	tests := []setTest{
		{0, 0, false},
		{2, 0, false},
		{2, 1, true},
		{7, 2, true},
		{7, 4, false},
	}

	for i, test := range tests {
		res := isSet(test.nb, test.idx)
		require.Equal(t, test.expected, res, "%d - failed: %v", i, test)
	}
}

func TestPartitionerRandomBin(t *testing.T) {
	n := 125
	reg := FakeRegistry(n)

	// try two different seeds
	s1 := []byte("Hello World")
	s2 := []byte("Sun is Shining")
	r1 := NewRandomBinPartitioner(1, reg, s1)
	r2 := NewRandomBinPartitioner(1, reg, s2)
	r3 := NewBinPartitioner(1, reg)

	ids1, more := r1.PickNextAt(6, 30)
	require.True(t, more)
	require.Equal(t, 30, len(ids1))
	ids2, more := r2.PickNextAt(6, 30)
	require.Equal(t, 30, len(ids2))
	require.True(t, more)

	ids11, more := r3.PickNextAt(6, 30)
	require.Equal(t, 30, len(ids11))
	require.True(t, more)

	// Given the size of the array, the probability to
	//  have exactly the same set is quite low, so this test
	//  should not be flaky
	require.NotEqual(t, ids1, ids2)
	require.NotEqual(t, ids1, ids11)

	ids3, ok := r1.PickNextAt(6, 30)
	require.True(t, ok)
	require.NotEqual(t, ids1, ids3)
}


func TestPartitionerPickNextAt(t *testing.T) {
	n := 32
	reg := FakeRegistry(n)
	r := NewRandomBinPartitioner(1, reg, []byte("Hello World"))
	//r := NewBinPartitioner(1, reg)
	ids1, res1 := r.PickNextAt(1, 10)
	require.True(t, res1)
	require.Equal(t, 1, len(ids1))

	_, res2 := r.PickNextAt(1, 1)
	require.False(t, res2)

	ids3, res3 := r.PickNextAt(2, 1)
	require.True(t, res3)
	require.Equal(t, 1, len(ids3))

	ids4, res4 := r.PickNextAt(2, 1)
	require.True(t, res4)
	require.Equal(t, 1, len(ids4))
	require.NotEqual(t, ids3[0], ids4[0])
}