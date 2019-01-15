package main

import (
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/p2p"
)

func main() {
	maker := p2p.AdaptorFunc(MakeP2P)
	maker = p2p.WithConnector(maker)
	maker = p2p.WithPostFunc(maker, func(handel.Registry, []p2p.Node) {
		time.Sleep(2 * time.Second)
	})

	p2p.Run(maker)
}
