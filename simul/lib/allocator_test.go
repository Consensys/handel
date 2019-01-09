package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocatorLinear(t *testing.T) {

	type allocTest struct {
		total    int
		offline  int
		expected []int
	}

	i := func(is ...int) []int {
		return is
	}
	r := func(from, to int) []int {
		arr := make([]int, to-from)
		for i := 0; i < len(arr); i++ {
			arr[i] = from + i
		}
		return arr
	}
	// golint not complaining of unused variable in this case
	i()

	var tests = []allocTest{
		{10, 0, r(0, 10)},
		{10, 1, r(1, 10)},
		{10, 5, i(1, 3, 5, 7, 9)},
		{10, 4, i(1, 3, 5, 7, 8, 9)},
	}
	allocator := new(linearAllocator)
	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		res := allocator.Allocate(test.total, test.offline)
		require.Equal(t, test.expected, res)
	}
}
