package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

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
		MaxTimeout:  "5m",
		Retrials:    1,
		Allocator:   "round",
		Simulation:  "handel",
		Debug:       0,
	}

	handel := &lib.HandelConfig{
		Period:      "20ms",
		UpdateCount: 1,
		NodeCount:   10,
		Timeout:     "100ms",
	}

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configDir := filepath.Join(dir, "configs")
	os.MkdirAll(configDir, 0777)

	// 4 instance per proc
	procF := getProcessF(4)

	thresholdIncScenario(configDir, defaultConf, handel)
	failingIncScenario(configDir, defaultConf, handel, procF)
	timeoutIncScenario(configDir, defaultConf, handel, procF)
}

func timeoutIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, procF func(int) int) {
	// just two failings scenario to see the effect of timeout on different
	// threshold
	failings := []float64{0.25, 0.49}
	timeouts := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}
	n := 2001
	thr := 0.99 // 99% of the ALIVE nodes
	var runs []lib.RunConfig
	for _, f := range failings {
		failing := thrF(f)(n)
		threshold := thrF(thr)(n - failing) // we want all alive nodes
		for _, t := range timeouts {
			handelConf := &lib.HandelConfig{
				Period:                     handel.Period,
				UpdateCount:                handel.UpdateCount,
				NodeCount:                  handel.NodeCount,
				Timeout:                    t.String(),
				UnsafeSleepTimeOnSigVerify: handel.UnsafeSleepTimeOnSigVerify,
			}
			run := lib.RunConfig{
				Nodes:     n,
				Threshold: threshold,
				Failing:   failing,
				Processes: procF(n),
				Handel:    handelConf,
			}
			runs = append(runs, run)
		}
	}

	defaultConf.Runs = runs
	fileName := "2000nodesTimeoutInc.toml"
	if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
		panic(err)
	}

}

func failingIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, procF func(int) int) {
	// just to see we dont have side effects when only waiting on 51% - since
	// it's the last step of handel
	thrs := []float64{0.51, 0.75}
	// various percentages  of failing nodes
	failings := []float64{0.01, 0.25, 0.49, 0.75}
	n := 2001
	var runs []lib.RunConfig
	for _, thr := range thrs {
		threshold := thrF(thr)(n)
		for _, fail := range failings {
			failing := thrF(fail)(n)
			run := lib.RunConfig{
				Nodes:     n,
				Threshold: threshold,
				Failing:   failing,
				Processes: procF(n),
				Handel:    handel,
			}
			runs = append(runs, run)
		}
	}
	defaultConf.Runs = runs
	fileName := "2000nodesFailingInc.toml"
	if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
		panic(err)
	}
}

func thresholdIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig) {
	// various threshold to use
	thrs := []float64{0.99, 0.75, 0.51, 0.30}
	for _, thr := range thrs {
		//nodeIncScenario(defaultConf, handel, "2000Nodes200Inst80.toml")
		nodesInc := scenarios.NewNodeInc(defaultConf, handel, 2001, 4, 0, thrF(thr))
		conf := nodesInc.Generate(400)
		fileName := fmt.Sprintf("2000nodes200inc%dthr.toml", int(thr*100))
		full := filepath.Join(dir, fileName)
		if err := conf.WriteTo(full); err != nil {
			panic(err)
		}
	}
}

func thrF(t float64) func(int) int {
	return func(n int) int {
		return scenarios.CalcThreshold(n, t)
	}
}

func getProcessF(instancePerProc int) func(int) int {
	return func(nodes int) int {
		nbProc := float64(nodes) / float64(instancePerProc)
		return int(math.Ceil(nbProc))
	}
}
