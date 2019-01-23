package main

import (
	"context"
	"fmt"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
)

// MakeP2P returns the constructor for the libp2p node
func MakeP2P(ctx context.Context, nodes lib.NodeList, ids []int, threshold int, opts p2p.Opts) (handel.Registry, []p2p.Node) {
	total := len(nodes)
	//pubsub.GossipSubHistoryLength = total
	//pubsub.GossipSubHistoryGossip = total
	//pubsub.GossipSubHeartbeatInterval = 500 * time.Millisecond
	cons := ctx.Value(p2p.CtxKey("Constructor")).(lib.Constructor)
	var router = getRouter(opts)
	var registry = P2PRegistry(make([]*P2PIdentity, total))
	var ns []p2p.Node
	var err error
	for _, node := range nodes {
		id := int(node.Identity.ID())
		registry[id], err = NewP2PIdentity(node.Identity, cons)
		if err != nil {
			fmt.Println("err: ", err)
			panic(err)
		}

		if p2p.IsIncluded(ids, id) {
			p2pNode, err := NewP2PNode(ctx, node, router, registry, cons, threshold)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
			//buff, _ := p2pNode.priv.SecretKey.MarshalBinary()
			//fmt.Printf(" ++ Make() adding p2pNode %s\n", hex.EncodeToString(buff[0:16]))
			ns = append(ns, p2pNode)
		}
	}
	return &registry, ns
}
