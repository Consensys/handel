package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var nb = NewWilffBitset

type bitsetTest struct {
	b           BitSet
	bitlength   int
	cardinality int
	setBits     []int
}

var tests = []bitsetTest{
	{
		nb(10), 10, 0, []int{},
	},
}

func TestBitSetWilff(t *testing.T) {
	for _, tt := range tests {
		require.Equal(t, tt.bitlength, tt.b.BitLength())
		require.Equal(t, tt.cardinality, tt.b.Cardinality())
		for _, idx := range tt.setBits {
			require.True(t, tt.b.Get(idx))
		}
	}
}
