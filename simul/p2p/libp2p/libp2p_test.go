package main

import (
	"sync"
	"testing"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
	"github.com/ConsenSys/handel/simul/p2p/test"
)

func TestP2P(t *testing.T) {
	t.Skip()
	n := 50
	thr := 15
	var opts = map[string]string{"Connector": "neighbor", "Count": "8"}
	maker := p2p.AdaptorFunc(MakeP2P)
	maker = p2p.WithConnector(maker)
	maker = p2p.WithPostFunc(maker, func(r handel.Registry, nodes []p2p.Node) {
		var wg sync.WaitGroup
		for _, n := range nodes {
			wg.Add(1)
			go func(n *P2PNode) {
				n.WaitAllSetup()
				wg.Done()
			}(n.(*P2PNode))
		}
		wg.Wait()
	})

	test.Aggregators(t, n, thr, maker, opts, lib.GetFreeTCPPort)

}
