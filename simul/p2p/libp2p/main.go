package main

import (
	"sync"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/p2p"
)

func main() {
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

	p2p.Run(maker)
}
