package main

import (
	"github.com/ConsenSys/handel/simul/confgenerator/scenarios"
	"github.com/ConsenSys/handel/simul/lib"
)

type configGen interface {
	Generate(step int) lib.Config
}

func main() {
	defaultConf := lib.Config{
		Network:     "udp",
		Curve:       "bn256",
		Encoding:    "gob",
		MonitorPort: 10000,
		MaxTimeout:  "2m",
		Retrials:    1,
		Allocator:   "linear",
		Simulation:  "handel",
		Debug:       0,
	}

	handel := &lib.HandelConfig{
		Period:      "20ms",
		UpdateCount: 1,
		NodeCount:   10,
		Timeout:     "100ms",
	}

	nodeIncScenario(defaultConf, handel, "2000Nodes200Inst80.toml")
}

func nodeIncScenario(defaultConf lib.Config, handel *lib.HandelConfig, fileName string) {
	nodesInc := scenarios.NewNodeInc(defaultConf, handel, 2001, 4, 0, scenarios.CalcThreshold80)
	conf := nodesInc.Generate(200)
	conf.WriteTo(fileName)
}
