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

func TestPartitionerBinIndexAtLevel(t *testing.T) {
	n := 13
	reg := FakeRegistry(n)

	type indexTest struct {
		partID int32
		id     int32
		level  int
		err    bool
		exp    int
	}

	var tests = []indexTest{
		// "left side" id should be same
		{5, 1, 3, false, 1},
		// "right side" should be shifted
		{1, 5, 3, false, 1},
		// invalid level
		{1, 1, 10, true, 1},
		// invalid id for this level
		{1, 5, 2, true, 1},
	}

	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		part := NewBinPartitioner(test.partID, reg)
		res, err := part.IndexAtLevel(test.id, test.level)
		if err != nil {
			if test.err {
				require.Error(t, err)
				continue
			}
			t.Fatal("expected no error", err)
		}
		require.Equal(t, test.exp, res)
	}
}

func TestPartitionerBinTreeCombine(t *testing.T) {
	n := 17
	reg := FakeRegistry(n)

	type combineTest struct {
		id    int32
		sigs  []*incomingSig
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
	pairs3 := incomingSigs(0, 2, 3)
	final4 := finalIncomingSig(4, 8)
	final4.ms.BitSet.Set(0, false)

	fullBs := NewWilffBitset(n)
	for i := 0; i < n; i++ {
		fullBs.Set(i, true)
	}

	var tests = []combineTest{
		// from last node
		{16, incomingSigs(0), 1, false, &MultiSignature{Signature: sig3, BitSet: bs0}},
		// error in the level requested
		{16, incomingSigs(0, 5), 3, true, &MultiSignature{Signature: sig3, BitSet: fullBs}},
		// contributions from last node + all previous nodes
		{16, incomingSigs(0, 5), 6, false, &MultiSignature{Signature: sig3, BitSet: fullBs}},
		// all good, we should have the first half of signature returned (
		{1, incomingSigs(0, 1, 2, 3), 4, false, &MultiSignature{Signature: sig3, BitSet: bs3}},
		// only one to combine
		{1, incomingSigs(2), 3, false, &MultiSignature{Signature: sig2, BitSet: bs2}},
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

	pairs3 := incomingSigs(0, 1, 2, 3, 4)
	// 4-1 -> because that's how you compute the size of a level
	// -1 -> to just spread out holes to other levels and leave this one still
	// having one contribution
	for i := 0; i < pow2(4-1)-1; i++ {
		pairs3[4].ms.BitSet.Set(i, false)
	}
	pairs3[3].ms.BitSet.Set(1, false)
	pairs3[3].ms.BitSet.Set(2, false)

	final4 := finalIncomingSig(4, n)
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
		sigs  []*incomingSig
		isErr bool
		exp   *MultiSignature
	}

	var tests = []combineTest{
		// from last node
		{16, incomingSigs(0), false, &MultiSignature{Signature: sig3, BitSet: bs(16)}},
		// error in the level requested
		{16, incomingSigs(0, 5), true, &MultiSignature{Signature: sig3, BitSet: bs()}},
		// contributions from last node + all previous nodes
		{16, incomingSigs(0, 5), false, &MultiSignature{Signature: sig3, BitSet: bs()}},

		// all good, we should have the first half of signature returned (
		{1, incomingSigs(0, 1, 2, 3), false, &MultiSignature{Signature: sig3, BitSet: bs3}},
		// only one to combine
		{1, incomingSigs(2), false, &MultiSignature{Signature: sig2, BitSet: bs2}},
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

func TestPartitionerBinLevels(t *testing.T) {
	type levelsTest struct {
		n      int
		id     int32
		levels []int
	}

	i := func(is ...int) []int {
		return is
	}

	tests := []levelsTest{
		{4, 1, i(1, 2)},
		{5, 1, i(1, 2, 3)},
		{5, 4, i(3)},
	}

	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		reg := FakeRegistry(test.n)
		ct := NewBinPartitioner(test.id, reg).(*binomialPartitioner)
		levels := ct.Levels()
		require.Equal(t, test.levels, levels)
		for _, lvl := range levels {
			_, err := ct.IdentitiesAt(lvl)
			require.NoError(t, err)
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
