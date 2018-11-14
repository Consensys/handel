package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
		{1, false, 0, 1},
		{2, false, 2, 4},
		{3, false, 4, 8},
		{4, false, 8, 16},
		{0, true, 0, 0},
		{7, true, 0, 0},
	}

	for _, test := range tests {
		_ids, err := ct.RangeAt(test.level)
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
