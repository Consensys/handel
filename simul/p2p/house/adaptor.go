package house

import (
	"context"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
)

// MakeGossip is an adaptor to be able to run simulation with the node
func MakeGossip(ctx context.Context, list lib.NodeList, ids []int, threshold int, opts p2p.Opts) (handel.Registry, []p2p.Node) {
	fanout := extractFanout(opts)
	period := extractPeriod(opts)
	reg := list.Registry()
	n := reg.Size()
	hnodes := make([]p2p.Node, n)
	var err error
	for i := 0; i < n; i++ {
		hnodes[i], err = NewNode(list[i], reg, fanout, period)
		if err != nil {
			panic(err)
		}
	}
	return reg, hnodes
}

// DefaultFanOut holds how many nodes does one contact at each update
var DefaultFanOut = 5

func extractFanout(o p2p.Opts) int {
	f, e := o.Int("Fanout")
	if !e {
		f = DefaultFanOut
	}
	return f
}

// DefaultPeriod holds the period of the update the node use
var DefaultPeriod = "300ms"

func extractPeriod(o p2p.Opts) time.Duration {
	s, e := o.String("Period")
	if !e {
		s = DefaultPeriod
	}
	t, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return t
}
