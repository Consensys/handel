package platform

import (
	"strconv"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

func defaultStats(c *lib.Config, i int, r *lib.RunConfig) *monitor.Stats {
	return DefaultStats(i, r.Nodes, r.Threshold, c.Network)
}

// DefaultStats returns default stats
func DefaultStats(run int, nodes int, threshold int, network string) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"run":       strconv.Itoa(run),
		"nodes":     strconv.Itoa(nodes),
		"threshold": strconv.Itoa(threshold),
		"network":   network,
	}, nil)
}
