package lib

import (
	"testing"

	"github.com/ConsenSys/handel"
	"github.com/stretchr/testify/require"
)

func fakeNodes(totalSize int, active int) []*Node {
	nodes := []*Node{}
	for i := 0; i < active; i++ {
		identity := handel.NewStaticIdentity(int32(i), "", nil)
		nodes = append(nodes, &Node{Identity: identity, Active: true})
	}
	for i := 0; i < totalSize-active; i++ {
		identity := handel.NewStaticIdentity(int32(i), "", nil)
		nodes = append(nodes, &Node{Identity: identity, Active: false})
	}
	return nodes
}

func TestNoByzantineDistribution(t *testing.T) {
	nbOfByz := 0
	totalNBOfNodes := 10
	err, nodes := distribute(nbOfByz, 9989, totalNBOfNodes, 8)
	require.Equal(t, err, nil)

	require.Equal(t, totalNBOfNodes, len(nodes))
	for _, n := range nodes {
		require.False(t, n.IsByzantine)
	}
}

func TestByzantineDistribution(t *testing.T) {
	nbOfByz := 3
	totalNBOfNodes := 10
	err, nodes := distribute(nbOfByz, 9989, totalNBOfNodes, 8)

	require.Equal(t, err, nil)

	require.Equal(t, totalNBOfNodes, len(nodes))
	byzCount := 0
	for _, n := range nodes {
		if n.IsByzantine {
			require.True(t, n.Active)
			byzCount++
		}
	}
	require.Equal(t, nbOfByz, byzCount)

	err, nodes2 := distribute(nbOfByz, 776, totalNBOfNodes, 8)
	require.NotEqual(t, nodes2, nodes)
	require.Equal(t, err, nil)

	err, _ = distribute(nbOfByz, 776, totalNBOfNodes, 2)
	require.Error(t, err)
}

func distribute(nbOfByz int, seed int64, nbOfnodes, active int) (error, []*Node) {
	dist := RandomDistribution(seed)
	nodes := fakeNodes(nbOfnodes, active)
	err := dist.Distribute(nodes, nbOfByz)
	if err != nil {
		return err, nil
	}
	return nil, nodes
}

func TestShuffle(t *testing.T) {
	seed := int64(399)
	orig, shuffled := shuffleNodes(10, seed)

	require.Equal(t, len(orig), len(shuffled))
	require.True(t, isUnique(shuffled))
	require.NotEqual(t, orig, shuffled)

	orig2, shuffled2 := shuffleNodes(10, 999)
	require.Equal(t, orig, orig2)
	require.NotEqual(t, shuffled, shuffled2)

	orig, shuffled = shuffleNodes(1, seed)
	require.Equal(t, orig, shuffled)

	orig, shuffled = shuffleNodes(0, seed)
	require.Equal(t, orig, shuffled)
}

func shuffleNodes(size int, seed int64) ([]*Node, []*Node) {
	nodes := fakeNodes(size, size)
	tmp := make([]*Node, len(nodes))
	copy(tmp, nodes)
	shuffle(nodes, seed)
	return tmp, nodes
}

func isUnique(nodes []*Node) bool {
	unique := make([]bool, len(nodes))
	for _, n := range nodes {
		if unique[n.ID()] {
			return false
		}
		unique[n.ID()] = true
	}
	return true
}
