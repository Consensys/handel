package platform

import (
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

func defaultStats(c *lib.Config, i int, r *lib.RunConfig) *monitor.Stats {
	return monitor.DefaultStats(i, r.Nodes, r.Threshold, c.Network)
}
