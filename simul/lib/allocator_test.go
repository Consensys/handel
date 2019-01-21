package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type PlatString string

func (p *PlatString) String() string {
	return string(*p)
}

func TestAllocatorRoundRobin(t *testing.T) {
	type allocTest struct {
		plats    int
		total    int
		offline  int
		expected map[string][]*NodeInfo
	}

	// create one platform from the integer
	p := func(n int) Platform {
		plat := PlatString(fmt.Sprintf("plat-%d", n))
		return &plat
	}
	// returns a list of platforms with increasing integer
	ps := func(nb int) []Platform {
		var arr []Platform
		for i := 0; i < nb; i++ {
			arr = append(arr, p(i))
		}
		return arr
	}

	// return a node info
	ni := func(id int, status bool) *NodeInfo {
		return &NodeInfo{ID: id, Active: status}
	}

	// return a map containing all nodes given for the given platform
	fp := func(p Platform, nodes ...*NodeInfo) map[string][]*NodeInfo {
		var out = make(map[string][]*NodeInfo)
		s := p.String()
		for _, n := range nodes {
			out[s] = append(out[s], n)
		}
		return out
	}

	// merge the different platform's maps together
	fps := func(plats ...map[string][]*NodeInfo) map[string][]*NodeInfo {
		var out = make(map[string][]*NodeInfo)
		for _, p := range plats {
			for s, nodes := range p {
				out[s] = nodes
			}
		}
		return out
	}

	fps()
	// golint not complaining of unused variable in this case

	var tests = []allocTest{
		// everything on the same platform
		{1, 5, 0, fp(p(0), ni(0, true), ni(1, true), ni(2, true), ni(3, true), ni(4, true))},
		// everything on two platform
		{2, 5, 0, fps(fp(p(0), ni(0, true), ni(2, true), ni(4, true)), fp(p(1), ni(1, true), ni(3, true)))},
		// 2 failing nodes on the same platform
		{1, 5, 2, fp(p(0), ni(0, false), ni(1, true), ni(2, true), ni(3, false), ni(4, true))},
		// 3 failing nodes on two different platform
		// 0-f, 1-t, 2-t, 3-f, 4-t, 5-t, 6-f
		// -> plat 0 => 0,2,4,6
		// -> plat 1 => 1,3,5
		//{2, 7, 3, fps(fp(p(0), ni(0, false), ni(2, true), ni(4, true), ni(6, false)),
		//fp(p(1), ni(1, true), ni(3, false), ni(5, true)))},
	}
	allocator := new(RoundRobin)
	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		fmt.Printf(" -- test %d -- \n", i)
		plats := ps(test.plats)
		res := allocator.Allocate(plats, test.total, test.offline)
		require.Equal(t, test.expected, res)
	}
}
