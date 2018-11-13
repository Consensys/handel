package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCandidateTreeRangeAt(t *testing.T) {
	n := 16
	reg := FakeRegistry(n)
	//ids := reg.(*arrayRegistry).ids
	ct := newCandidateTree(1, reg)

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
		_ids, err := ct.IdentitiesAt(test.level)
		if test.isErr {
			require.Error(t, err)
		}
		min, max, err := ct.RangeAt(test.level)
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
