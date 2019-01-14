package p2p

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/bn256"
	"github.com/ConsenSys/handel/simul/lib"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/require"
)

func TestAggregator(t *testing.T) {
	n := 50
	nbOutgoing := 3
	connector := NewNeighborConnector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := pubsub.NewFloodSub
	ids, nodes := FakeSetup(ctx, n, nbOutgoing, connector, r)
	registry := P2PRegistry(ids)
	cons := bn256.NewConstructor()
	aggregators := make([]*Aggregator, n)
	time.Sleep(1 * time.Second)
	for i := 0; i < n; i++ {
		sig, err := nodes[i].priv.SecretKey.Sign(lib.Message, nil)
		require.NoError(t, err)
		agg := NewAggregator(nodes[i], &registry, cons, n, sig)
		aggregators[i] = agg
		go agg.Start()
	}

	var wg sync.WaitGroup
	for _, agg := range aggregators {
		wg.Add(1)
		go func(a *Aggregator) {
			sig := <-a.FinalMultiSignature()
			err := handel.VerifyMultiSignature(lib.Message, sig, &registry, cons)
			require.NoError(t, err)
			wg.Done()
		}(agg)
	}
	wg.Wait()
}
