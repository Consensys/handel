package platform

import (
	"strconv"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

func defaultStats(c *lib.Config, i int, r *lib.RunConfig) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"run":       strconv.Itoa(i),
		"nodes":     strconv.Itoa(r.Nodes),
		"threshold": strconv.Itoa(r.Threshold),
		"network":   c.Network,
	}, nil)
}
