package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionerBinTreeCombine(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	ct := newBinTreePartition(1, reg)

	var mkSigPair = func(level int) *sigPair {
		return &sigPair{
			level: byte(level),
			ms:    fullSig(level),
		}
	}

	var sigPairs = func(lvls ...int) []*sigPair {
		s := make([]*sigPair, len(lvls))
		for i, lvl := range lvls {
			s[i] = mkSigPair(lvl)
		}
		return s
	}

	type combineTest struct {
		sigs []*sigPair
		exp  *sigPair
	}

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

	var tests = []combineTest{
		// all good, we should have the first half of signature returned (
		{sigPairs(0, 1, 2, 3), &sigPair{level: 3, ms: &MultiSignature{Signature: sig3, BitSet: bs3}}},
		// only one to combine
		{sigPairs(2), &sigPair{level: 2, ms: &MultiSignature{Signature: sig2, BitSet: bs2}}},
		{nil, nil},
		// with holes
		{pairs3, final4},
	}

	for i, test := range tests {
		t.Logf(" -- test %d -- ", i)
		res := ct.Combine(test.sigs, NewWilffBitset)
		if test.exp == nil {
			require.Nil(t, res)
			continue
		}
		require.Equal(t, test.exp.ms.Signature, res.ms.Signature)
		require.Equal(t, test.exp.ms.BitSet.BitLength(), res.ms.BitSet.BitLength())
		bs1 := test.exp.ms.BitSet
		bs2 := res.ms.BitSet
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
		ct := newBinTreePartition(1, reg)
		require.Equal(t, test.exp, ct.MaxLevel())
	}
}

func TestPartitionerBinTreePickNextAt(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	ct := newBinTreePartition(1, reg)

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
	ct := newBinTreePartition(1, reg).(*binTreePartition)

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
