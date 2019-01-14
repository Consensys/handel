package p2p

import (
	"context"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
)

// Adaptor is an interface that gives back the registry and nodes from the
// list of node read from the simulation registry file
type Adaptor interface {
	Make(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node)
}

// AdaptorFunc returns an Adaptor out of a func
type AdaptorFunc func(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node)

// Make implements the Adaptor interface
func (a AdaptorFunc) Make(ctx context.Context, nodes []*lib.Node, ids []int, opts map[string]string) (handel.Registry, []Node) {
	return a(ctx, nodes, ids, opts)
}
