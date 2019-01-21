package scenarios

import "github.com/ConsenSys/handel/simul/lib"

type NodesInc struct {
	defaultConf   lib.Config
	defaultHandel *lib.HandelConfig
	maxNodes      int
	increment     int
	failing       int
	calcThreshold func(n int) int
}

func NewNodeInc(
	defaultConf lib.Config,
	handel *lib.HandelConfig,
	maxNodes,
	increment,
	failing int,
	calcThreshold func(int) int) NodesInc {

	return NodesInc{
		defaultConf:   defaultConf,
		defaultHandel: handel,
		maxNodes:      maxNodes,
		increment:     increment,
		failing:       failing,
		calcThreshold: calcThreshold,
	}
}

func (n NodesInc) Generate(proc int, nodesCt []int) lib.Config {
	var runs []lib.RunConfig
	for _, nodeCt := range nodesCt{
		run := lib.RunConfig{
			Nodes:     nodeCt,
			Threshold: n.calcThreshold(nodeCt),
			Failing:   n.failing,
			Processes: proc,
			Handel:    n.defaultHandel,
		}
		runs = append(runs, run)
	}
	n.defaultConf.Runs = runs
	return n.defaultConf
}
