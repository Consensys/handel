package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// MakeP2P returns the constructor for the libp2p node
func MakeP2P(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []p2p.Node) {
	total := len(nodes)
	pubsub.GossipSubHistoryLength = total
	pubsub.GossipSubHistoryGossip = total
	pubsub.GossipSubHeartbeatInterval = 500 * time.Millisecond
	var router = getRouter(opts)
	var registry = P2PRegistry(make([]*P2PIdentity, total))
	var ns = make([]p2p.Node, 0, len(ids))
	var err error
	for id, node := range nodes {
		registry[id], err = NewP2PIdentity(node.Identity)
		if err != nil {
			panic(err)
		}

		if p2p.IsIncluded(ids, id) {
			p2pNode, err := NewP2PNode(ctx, node, router, registry)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
			ns = append(ns, p2pNode)
		}
	}
	return &registry, ns
}
