package test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/bn256/cf"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
	"github.com/stretchr/testify/require"
)

var defaultResendP = 1 * time.Second

// Aggregators tests if a node's implementation works out with the aggregator
// logic before using it in simulation
func Aggregators(t *testing.T, n, thr int, a p2p.Adaptor, opts p2p.Opts, getPort func() int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, ids := fakeSetup(n, getPort)
	cons := lib.NewSimulConstructor(bn256.NewConstructor())
	ctx = context.WithValue(ctx, p2p.CtxKey("Constructor"), cons)
	reg, p2pNodes := a.Make(ctx, nodes, ids, thr, opts)
	aggregators := p2p.MakeAggregators(ctx, cons, p2pNodes, reg, thr, defaultResendP)

	var wg sync.WaitGroup
	var counter int32
	for _, agg := range aggregators {
		wg.Add(1)
		go func(a *p2p.Aggregator) {
			go a.Start()
			sig := <-a.FinalMultiSignature()
			require.True(t, sig.Cardinality() >= thr)
			err := handel.VerifyMultiSignature(lib.Message, sig, reg, cons.Handel())
			require.NoError(t, err)
			atomic.AddInt32(&counter, 1)
			fmt.Printf(" -- node %d finished, state %d/%d\n", a.Node.Identity().ID(), atomic.LoadInt32(&counter), reg.Size())
			wg.Done()
		}(agg)
	}
	wg.Wait()
}

func fakeSetup(n int, getPort func() int) (lib.NodeList, []int) {
	ids := make([]int, n)
	addresses := make([]string, n)
	for i := 0; i < n; i++ {
		// base := 40000
		//	port := base + i
		//port := lib.GetFreeTCPPort()
		port := getPort()
		address := "127.0.0.1:" + strconv.Itoa(port)
		addresses[i] = address
	}
	nodes := lib.GenerateNodes(lib.NewSimulConstructor(bn256.NewConstructor()), addresses)
	for i, nodes := range nodes {
		ids[i] = int(nodes.Identity.ID())
	}
	return nodes, ids
}
