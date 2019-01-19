package scenarios

import "github.com/ConsenSys/handel/simul/lib"

type nodesInc struct {
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
	calcThreshold func(int) int) nodesInc {

	return nodesInc{
		defaultConf:   defaultConf,
		defaultHandel: handel,
		maxNodes:      maxNodes,
		increment:     increment,
		failing:       failing,
		calcThreshold: calcThreshold,
	}
}

func (n nodesInc) Generate(step int) lib.Config {
	proc := 1
	var runs []lib.RunConfig
	for i := n.increment; i < n.maxNodes; i = i + n.increment {
		run := lib.RunConfig{
			Nodes:     i,
			Threshold: n.calcThreshold(i),
			Failing:   n.failing,
			Processes: proc,
			Handel:    n.defaultHandel,
		}
		proc++
		if i%step == 0 {
			runs = append(runs, run)
		}
	}
	n.defaultConf.Runs = runs
	return n.defaultConf
}
