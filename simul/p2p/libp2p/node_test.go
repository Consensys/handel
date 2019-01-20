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
	"github.com/ConsenSys/handel/simul/p2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/require"
)

func TestGossipMeshy(t *testing.T) {
	n := 50
	nbOutgoing := 3
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connector := p2p.NewNeighborConnector()
	//connector := NewRandomConnector()
	r := pubsub.NewGossipSub
	_, nodes := FakeSetup(ctx, n, nbOutgoing, connector, r)

	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n *P2PNode) {
			n.WaitAllSetup()
			wg.Done()
		}(n)
	}
	wg.Wait()

	time.Sleep(1 * time.Second)

	// broadcast
	for i := 0; i < 50; i++ {
		sender := nodes[rand.Intn(n)]
		packet := &handel.Packet{Origin: int32(i)}
		fmt.Println("trial", i, "from node", sender.handelID)
		sender.Diffuse(packet)
		for j, n := range nodes {
			select {
			case p := <-n.Next():
				require.Equal(t, packet.Origin, p.Origin)
				fmt.Println("received from ", j)
			case <-time.After(1 * time.Second):
				t.FailNow()
			}
		}
	}

}

func FakeSetup(ctx context.Context, n int, max int, c p2p.Connector, r NewRouter) ([]*P2PIdentity, []*P2PNode) {
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
		p2pNodes[i], err = NewP2PNode(ctx, node, r, p2pIDs)
		if err != nil {
			panic(err)
		}
	}

	registry := P2PRegistry(p2pIDs)
	for _, n := range p2pNodes {
		c.Connect(n, &registry, max)
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
