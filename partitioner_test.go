package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionerBinTreeCombine(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	ct := NewBinPartitioner(1, reg)

	type combineTest struct {
		sigs  []*sigPair
		level int
		exp   *sigPair
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

	// final signature should have level 4 and bitlength 8
	// with first bit set to false
	pairs3 := sigPairs(0, 2, 3)
	final4 := finalSigPair(4, 8)
	final4.ms.BitSet.Set(0, false)

	var tests = []combineTest{
		// all good, we should have the first half of signature returned (
		{sigPairs(0, 1, 2, 3), 4, &sigPair{level: 4, ms: &MultiSignature{Signature: sig3, BitSet: bs3}}},
		// only one to combine
		{sigPairs(2), 3, &sigPair{level: 3, ms: &MultiSignature{Signature: sig2, BitSet: bs2}}},
		{nil, 0, nil},
		// with holes
		{pairs3, 4, final4},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		res := ct.Combine(test.sigs, test.level, NewWilffBitset)
		if test.exp == nil {
			require.Nil(t, res)
			continue
		}
		require.Equal(t, test.exp.level, res.level)
		require.Equal(t, test.exp.ms.Signature, res.ms.Signature)
		require.Equal(t, test.exp.ms.BitSet.BitLength(), res.ms.BitSet.BitLength())
		expSize, _ := ct.Size(int(test.exp.level))
		require.Equal(t, expSize, res.ms.BitSet.BitLength())

		bs1 := test.exp.ms.BitSet
		bs2 := res.ms.BitSet
		for i := 0; i < bs1.BitLength(); i++ {
			require.Equal(t, bs1.Get(i), bs2.Get(i))
		}
	}
}

func TestPartitionerBinTreeCombineFull(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	ct := NewBinPartitioner(1, reg)

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

	type combineTest struct {
		sigs []*sigPair
		exp  *MultiSignature
	}

	var tests = []combineTest{
		// all good, we should have the first half of signature returned (
		{sigPairs(0, 1, 2, 3), &MultiSignature{Signature: sig3, BitSet: bs3}},
		// only one to combine
		{sigPairs(2), &MultiSignature{Signature: sig2, BitSet: bs2}},
		{nil, nil},
		// with holes
		{pairs3, final4.ms},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
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
	n := 16
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
	n := 16
	reg := FakeRegistry(n)
	ct := NewBinPartitioner(1, reg).(*binomialPartitioner)

	type rangeTest struct {
		level int
		isErr bool
		from  int
		to    int
	}

	tests := []rangeTest{
		{0, false, 1, 2},
		{1, false, 0, 1},
		{2, false, 2, 4},
		{3, false, 4, 8},
		{4, false, 8, 16},
		{7, true, 0, 0},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		_ids, err := ct.IdentitiesAt(test.level)
		if test.isErr {
			require.Error(t, err)
		}
		min, max, err := ct.rangeLevel(test.level)
		if test.isErr {
			require.Error(t, err)
			continue
		}
		require.Equal(t, min, test.from)
		require.Equal(t, max, test.to)

		expected, ok := reg.Identities(test.from, test.to)
		require.True(t, ok)
		require.Equal(t, expected, _ids)
	}
}

func TestPartitionerBinTreeRangeAtInverse(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	ct := NewBinPartitioner(1, reg).(*binomialPartitioner)

	type rangeTest struct {
		level int
		isErr bool
		from  int
		to    int
	}

	tests := []rangeTest{
		{0, false, 1, 1},
		{1, false, 1, 2},
		{2, false, 0, 2},
		{3, false, 0, 4},
		{4, false, 0, 8},
		{5, false, 0, 16},
		{7, true, 0, 0},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
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
	n := 16
	reg := FakeRegistry(n)

	// try two different seeds
	s1 := []byte("Hello World")
	s2 := []byte("Sun is Shining")
	r1 := NewRandomBinPartitioner(1, reg, s1)
	r2 := NewRandomBinPartitioner(1, reg, s2)
	r3 := NewBinPartitioner(1, reg)

	ids1, more := r1.PickNextAt(3, 5)
	require.True(t, more)
	ids2, more := r2.PickNextAt(3, 5)
	require.True(t, more)

	ids11, more := r3.PickNextAt(3, 5)
	require.True(t, more)

	require.NotEqual(t, ids1, ids2)
	require.NotEqual(t, ids1, ids11)

	ids3, ok := r1.PickNextAt(3, 5)
	require.True(t, ok)
	require.NotEqual(t, ids1, ids3)
}
