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
		Allocator:   "random",
		Simulation:  "handel",
		Debug:       0,
	}

	handel := &lib.HandelConfig{
		Period:      "10ms",
		UpdateCount: 1,
		NodeCount:   10,
		Timeout:     "50ms",
	}

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configDir := filepath.Join(dir, "generated_configs")
	os.MkdirAll(configDir, 0777)

	// 2 handel per instance
	fixedProcesses := getProcessF(2)
	//  1 handel per instance up until 2000 where we go with 2
	//adaptiveProcesses := adaptiveGetProcessF(2, 2000)
	// some scenario can add higher nodes
	//baseNodes := []int{100, 300, 500, 1000, 1500, 2000, 2500,4000 3000, 4000}
	baseNodes := []int{100, 300, 500, 1000, 1500, 2000}

	// one threshold increase with fixed
	thresholdIncScenario2(configDir, defaultConf, handel, baseNodes, fixedProcesses)
	failingIncScenario(configDir, defaultConf, handel, baseNodes, fixedProcesses)
	/* // one threshold increase with adaptive process ->*/
	//thresholdIncScenario2(configDir, defaultConf, handel, baseNodes, adaptiveProcesses)
	//// we can go to 2000 hopefully with failing nodes
	//// since we go high, we need adaptive
	nsquareScenario(configDir, defaultConf, handel, baseNodes, fixedProcesses)
	libp2pScenario(configDir, defaultConf, handel, baseNodes, fixedProcesses)
	timeoutIncScenario(configDir, defaultConf, handel, baseNodes, fixedProcesses)
	periodIncScenario(configDir, defaultConf, handel, baseNodes, fixedProcesses)
}

func libp2pScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	oldSimul := defaultConf.Simulation
	defer func() { defaultConf.Simulation = oldSimul }()

	defaultConf.Simulation = "p2p/libp2p"
	//nodes := append(baseNodes, 3000, 4000)
	nodes := baseNodes
	thresholds := []float64{0.51, 0.75, 0.99}
	for _, thr := range thresholds {
		for _, verify := range []string{"1"} {
			var runs []lib.RunConfig
			for _, n := range nodes {
				run := lib.RunConfig{
					Nodes:     n,
					Threshold: thrF(thr)(n),
					Failing:   0,
					Processes: procF(n),
					Handel:    handel,
					Extra: map[string]string{
						"AggAndVerify": verify,
					},
				}
				fmt.Println(" n = ", n, " => process = ", procF(n))
				runs = append(runs, run)
			}
			defaultConf.Runs = runs
			fileName := fmt.Sprintf("4000node_Libp2pInc_%dthr_agg%s.toml", int(thr*100), verify)
			if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
				panic(err)
			}

		}
	}
}

func nsquareScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	oldSimul := defaultConf.Simulation
	defer func() { defaultConf.Simulation = oldSimul }()

	defaultConf.Simulation = "p2p/udp"
	//nodes := append(baseNodes, 3000, 4000)
	nodes := baseNodes
	thresholds := []float64{0.51, 0.75, 0.99}
	for _, thr := range thresholds {
		for _, verify := range []string{"1"} {
			var runs []lib.RunConfig
			for _, n := range nodes {
				run := lib.RunConfig{
					Nodes:     n,
					Threshold: thrF(thr)(n),
					Failing:   0,
					Processes: procF(n),
					Handel:    handel,
					Extra: map[string]string{
						"AggAndVerify": verify,
					},
				}
				runs = append(runs, run)
			}
			defaultConf.Runs = runs
			fileName := fmt.Sprintf("4000node_nsquareInc_%dthr_agg%s.toml", int(thr*100), verify)
			if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
				panic(err)
			}

		}
	}
}

// periodIncScenario increases the "update" period
func periodIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	periods := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
	}
	failing := 0.25
	thr := 0.99 // 99% of the ALIVE nodes
	for _, p := range periods {
		var runs []lib.RunConfig
		for _, nodes := range baseNodes {
			failing := thrF(failing)(nodes)
			threshold := thrF(thr)(nodes - failing) // we want all alive nodes
			handelConf := &lib.HandelConfig{
				Period:                     p.String(),
				UpdateCount:                handel.UpdateCount,
				NodeCount:                  handel.NodeCount,
				Timeout:                    handel.Timeout,
				UnsafeSleepTimeOnSigVerify: handel.UnsafeSleepTimeOnSigVerify,
			}
			run := lib.RunConfig{
				Nodes:     nodes,
				Threshold: threshold,
				Failing:   failing,
				Processes: procF(nodes),
				Handel:    handelConf,
			}
			runs = append(runs, run)
		}
		defaultConf.Runs = runs
		fileName := fmt.Sprintf("2000node_%dperiod_%dfail_%dthr.toml", toMilli(p), int(failing*100), int(thr*100))
		if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
			panic(err)
		}

	}

}

// scenario that increases the timeout with different failing number of nodes -
// threshold is fixed to 0.99 * alive node
func timeoutIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	failings := []float64{0.25}
	timeouts := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}
	thr := 0.99 // 99% of the ALIVE nodes
	for _, t := range timeouts {
		var runs []lib.RunConfig
		for _, node := range baseNodes {
			for _, f := range failings {
				failing := thrF(f)(node)
				threshold := thrF(thr)(node - failing) // we want all alive nodes
				handelConf := &lib.HandelConfig{
					Period:                     handel.Period,
					UpdateCount:                handel.UpdateCount,
					NodeCount:                  handel.NodeCount,
					Timeout:                    t.String(),
					UnsafeSleepTimeOnSigVerify: handel.UnsafeSleepTimeOnSigVerify,
				}
				run := lib.RunConfig{
					Nodes:     node,
					Threshold: threshold,
					Failing:   failing,
					Processes: procF(node),
					Handel:    handelConf,
				}
				runs = append(runs, run)
			}
		}
		defaultConf.Runs = runs
		fileName := fmt.Sprintf("2000nodes_%dtimeout_%dthr.toml", toMilli(t), int(thr*100))
		if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
			panic(err)
		}

	}

}

// failingIncScenario increases the number of failing nodes with two different
// threshold.
func failingIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	thr := 0.51
	// various percentages  of failing nodes
	failings := []float64{0.01, 0.25, 0.49}
	for _, fail := range failings {
		var runs []lib.RunConfig
		for _, nodes := range baseNodes {
			failing := thrF(fail)(nodes)
			//fmt.Printf("failing %d for %d nodes\n", failing, nodes)
			threshold := thrF(thr)(nodes - failing)
			run := lib.RunConfig{
				Nodes:     nodes,
				Threshold: threshold,
				Failing:   failing,
				Processes: procF(nodes),
				Handel:    handel,
			}
			runs = append(runs, run)
		}
		defaultConf.Runs = runs
		fileName := fmt.Sprintf("2000nodes_%dfail_%dthr.toml", int(fail*100), int(thr*100))
		if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
			panic(err)
		}
	}
}

func thresholdIncScenario2(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int, procF func(int) int) {
	// just to see we dont have side effects when only waiting on 51% - since
	// it's the last step of handel
	thrs := []float64{0.51, 0.75, 0.99}
	nodeList := baseNodes // append(baseNodes, []int{3000, 4000}...)
	baseNodes = append(baseNodes, 5000, 6000)
	for _, thr := range thrs {
		var runs []lib.RunConfig
		for _, nodes := range nodeList {
			threshold := thrF(thr)(nodes)
			run := lib.RunConfig{
				Nodes:     nodes,
				Threshold: threshold,
				Failing:   0,
				Processes: procF(nodes),
				Handel:    handel,
			}
			runs = append(runs, run)
		}
		defaultConf.Runs = runs
		fileName := fmt.Sprintf("4000nodes_%dthr.toml", int(thr*100))
		if err := defaultConf.WriteTo(filepath.Join(dir, fileName)); err != nil {
			panic(err)
		}
	}

}

// thresholdIncScenario tries different number of nodes with a list of different
// threshold to use
/*func thresholdIncScenario(dir string, defaultConf lib.Config, handel *lib.HandelConfig, baseNodes []int) {*/

//// do we want to output in one file or not
//oneFile := false
//// various threshold to use
//thrs := []float64{0.99, 0.75, 0.51}
//for _, thr := range thrs {
////nodeIncScenario(defaultConf, handel, "2000Nodes200Inst80.toml")
//nodesInc := scenarios.NewNodeInc(defaultConf, handel, 3001, 4, 0, thrF(thr))
//conf := nodesInc.Generate(2, append(baseNodes, []int{3000, 4000}))
//if oneFile {
//defaultConf.Runs = append(defaultConf.Runs, conf.Runs...)
//} else {
//fileName := fmt.Sprintf("test_0failing_%dthr.toml", int(thr*100))
//full := filepath.Join(dir, fileName)
//if err := conf.WriteTo(full); err != nil {
//panic(err)
//}
//}
//}

//if oneFile {
//fileName := fmt.Sprintf("2000nodes200ThresholdInc.toml")
//full := filepath.Join(dir, fileName)
//if err := defaultConf.WriteTo(full); err != nil {
//panic(err)
//}
//}
//}

func thrF(t float64) func(int) int {
	return func(n int) int {
		return scenarios.CalcThreshold(n, t)
	}
}

func adaptiveGetProcessF(instancePerProc, threshold int) func(int) int {
	return func(nodes int) int {
		if nodes >= threshold {
			nbProc := float64(nodes) / float64(instancePerProc)
			return int(math.Ceil(nbProc))
		}
		return nodes
	}
}

func getProcessF(instancePerProc int) func(int) int {
	return func(nodes int) int {
		nbProc := float64(nodes) / float64(instancePerProc)
		return int(math.Ceil(nbProc))
	}
}

func toMilli(t time.Duration) int64 {
	return t.Nanoseconds() / 1e6
}
