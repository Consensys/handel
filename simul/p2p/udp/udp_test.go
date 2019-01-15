package main

import (
	"testing"

	"github.com/ConsenSys/handel/simul/p2p"
	"github.com/ConsenSys/handel/simul/p2p/test"
)

func TestUDP(t *testing.T) {
	n := 20
	thr := 15
	maker := p2p.AdaptorFunc(MakeUDP)

	test.Aggregators(t, n, thr, maker, nil)

}
