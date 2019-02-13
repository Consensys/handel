package p2p

import (
	"context"
	"fmt"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
)

// Adaptor is an interface that gives back the registry and nodes from the
// list of node read from the simulation registry file
type Adaptor interface {
	Make(ctx context.Context, nodes lib.NodeList, ids []int, threshold int, opts Opts) (handel.Registry, []Node)
}

// AdaptorFunc returns an Adaptor out of a func
type AdaptorFunc func(ctx context.Context, nodes lib.NodeList, ids []int, threshold int, opts Opts) (handel.Registry, []Node)

// Make implements the Adaptor interface
func (a AdaptorFunc) Make(ctx context.Context, nodes lib.NodeList, ids []int, threshold int, opts Opts) (handel.Registry, []Node) {
	return a(ctx, nodes, ids, threshold, opts)
}

// WithConnector returns a Adaptor that also connects the nodes according to the
// connector specified in the opts.
func WithConnector(a AdaptorFunc) AdaptorFunc {
	return func(ctx context.Context, lnodes lib.NodeList, ids []int, threshold int, opts Opts) (handel.Registry, []Node) {
		reg, nodes := a(ctx, lnodes, ids, threshold, opts)
		connector, count := ExtractConnector(opts)
		for _, node := range nodes {
			err := connector.Connect(node, reg, count)
			if err != nil {
				fmt.Println("err : ", err)
				panic(err)
			}
		}
		return reg, nodes
	}
}

// WithPostFunc returns an Adaptor that executes a function after the given
// Adaptor
func WithPostFunc(a AdaptorFunc, fn func(reg handel.Registry, ns []Node)) AdaptorFunc {
	return func(ctx context.Context, lnodes lib.NodeList, ids []int, threshold int, opts Opts) (handel.Registry, []Node) {
		reg, nodes := a(ctx, lnodes, ids, threshold, opts)
		fn(reg, nodes)
		return reg, nodes
	}
}
