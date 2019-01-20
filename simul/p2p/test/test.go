package test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/bn256"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
	"github.com/stretchr/testify/require"
)

var defaultResendP = 1 * time.Second

// Aggregators tests if a node's implementation works out with the aggregator
// logic before using it in simulation
func Aggregators(t *testing.T, n, thr int, a p2p.Adaptor, opts p2p.Opts) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, ids := fakeSetup(n)
	reg, p2pNodes := a.Make(ctx, nodes, ids, opts)
	cons := lib.NewSimulConstructor(bn256.NewConstructor())
	aggregators := p2p.MakeAggregators(ctx, cons, p2pNodes, reg, thr, defaultResendP)

	var wg sync.WaitGroup
	for _, agg := range aggregators {
		wg.Add(1)
		go func(a *p2p.Aggregator) {
			go a.Start()
			sig := <-a.FinalMultiSignature()
			require.True(t, sig.Cardinality() >= thr)
			err := handel.VerifyMultiSignature(lib.Message, sig, reg, cons.Handel())
			require.NoError(t, err)
			wg.Done()
		}(agg)
	}
	wg.Wait()
}

func fakeSetup(n int) (lib.NodeList, []int) {
	ids := make([]int, n)
	base := 2000
	addresses := make([]string, n)
	for i := 0; i < n; i++ {
		port := base + i
		address := "127.0.0.1:" + strconv.Itoa(port)
		addresses[i] = address
	}
	nodes := lib.GenerateNodes(lib.NewSimulConstructor(bn256.NewConstructor()), addresses)
	for i, nodes := range nodes {
		ids[i] = int(nodes.Identity.ID())
	}
	return nodes, ids
}
