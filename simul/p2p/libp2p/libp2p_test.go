package main

import (
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/p2p"
	"github.com/ConsenSys/handel/simul/p2p/test"
)

func TestP2P(t *testing.T) {
	n := 20
	thr := 15
	var opts = map[string]string{"Connector": "neighbor", "Count": "5"}
	maker := p2p.AdaptorFunc(MakeP2P)
	maker = p2p.WithConnector(maker)
	maker = p2p.WithPostFunc(maker, func(handel.Registry, []p2p.Node) {
		time.Sleep(2 * time.Second)
	})

	test.Aggregators(t, n, thr, maker, opts)

}
