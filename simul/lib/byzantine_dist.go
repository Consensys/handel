package lib

import (
	"errors"
	"math/rand"
)

type ByzantineDistribution interface {
	Distribute(nodes []*Node, numberOfByzantineNodes int) error
}

type randomDistribution struct {
	seed int64
}

func RandomDistribution(seed int64) ByzantineDistribution {
	return &randomDistribution{seed}
}

func (rd *randomDistribution) Distribute(nodes []*Node, numberOfByzantineNodes int) error {
	var activeNodes []*Node
	for _, n := range nodes {
		if n.Active {
			activeNodes = append(activeNodes, n)
		}
	}

	if len(activeNodes) < numberOfByzantineNodes {
		return errors.New("Not enough active nodes")
	}

	shuffle(activeNodes, rd.seed)
	for i := 0; i < numberOfByzantineNodes; i++ {
		activeNodes[i].IsByzantine = true
	}
	return nil
}

func shuffle(vals []*Node, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}
