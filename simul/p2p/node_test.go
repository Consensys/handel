package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
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
		agg := NewAggregator(nodes[i], &registry, cons, n)
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

func TestGossipMeshy(t *testing.T) {
	n := 50
	nbOutgoing := 3
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connector := NewNeighborConnector()
	//connector := NewRandomConnector()
	r := pubsub.NewGossipSub
	_, nodes := FakeSetup(ctx, n, nbOutgoing, connector, r)

	time.Sleep(1 * time.Second)

	// broadcast
	msg := []byte("bloublou")
	for i := 0; i < 50; i++ {
		sender := nodes[rand.Intn(n)]
		fmt.Println("trial", i, "from node", sender.handelID)
		require.NoError(t, sender.Gossip(msg))
		for j, n := range nodes {
			rcvd, err := n.Next()
			fmt.Println("received from ", j)
			require.NoError(t, err)
			require.Equal(t, msg, rcvd)
		}
	}

}

func FakeSetup(ctx context.Context, n int, max int, c Connector, r NewRouter) ([]*P2PIdentity, []*P2PNode) {
	base := 2000
	addresses := make([]string, n)
	for i := 0; i < n; i++ {
		port := base + i
		address := "127.0.0.1:" + strconv.Itoa(port)
		addresses[i] = address
	}
	nodes := lib.GenerateNodes(lib.NewSimulConstructor(bn256.NewConstructor()), addresses)

	p2pNodes := make([]*P2PNode, n)
	p2pIDs := make([]*P2PIdentity, n)
	var err error
	for i := 0; i < n; i++ {
		node := nodes[i]
		p2pIDs[i], err = NewP2PIdentity(node.Identity)
		if err != nil {
			panic(err)
		}
		p2pNodes[i], err = NewP2PNode(ctx, node, r)
		if err != nil {
			panic(err)
		}
	}

	for _, n := range p2pNodes {
		c.Connect(n, p2pIDs, max)
	}

	return p2pIDs, p2pNodes
}

func connectRandom(t *testing.T, out int, nodes []*P2PNode, ids []*P2PIdentity) {
	n := len(nodes)
	for i := 0; i < n; i++ {
		node := nodes[i]
		randomIDs := rand.Perm(n)
		chosen := 0
		for _, id := range randomIDs {
			if id == i {
				continue
			}
			identity := ids[id]
			require.NoError(t, node.Connect(identity))
			chosen++
			if chosen >= out {
				break
			}
		}
	}
}

func connectNeighbors(t *testing.T, out int, nodes []*P2PNode, ids []*P2PIdentity) {
	n := len(nodes)
	for i := 0; i < n; i++ {
		node := nodes[i]
		chosen := 0
		j := i + 1
		for chosen < out {
			if j == n {
				j = 0
			}
			if j == i {
				continue
			}
			identity := ids[j]
			require.NoError(t, node.Connect(identity))
			chosen++
			j++

		}
	}

}
