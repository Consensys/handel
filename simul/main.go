// This package can launches a Handel simulation. It works the following way:
// 1. Read the config TOML file
// 2. Construct the right platform from the flag
// 3. Gives the Config to the Platform
// 4. Run the platform's Run
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
	"github.com/ConsenSys/handel/simul/platform"
)

var configFlag = flag.String("config", "", "TOML encoded config file")
var platformFlag = flag.String("platform", "", "name of the platform to run on")
var runTimeout = flag.Duration("run-timeout", 2*time.Minute, "timeout of a given run")

var resultsDir string

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	resultsDir = path.Join(currentDir, "results")
}
func main() {
	flag.Parse()

	// load configs
	c := lib.LoadConfig(*configFlag)
	plat := platform.NewPlatform(*platformFlag)
	if err := plat.Configure(c); err != nil {
		panic(err)
	}

	// preparation phase
	plat.Cleanup()
	os.MkdirAll(resultsDir, 0777)
	csvName := strings.Replace(filepath.Base(*configFlag), ".toml", ".csv", 1)
	csvName = filepath.Join(resultsDir, csvName)
	csvFile, err := os.Create(csvName)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	timeout := *runTimeout * time.Duration(c.Retrials)

	// running rounds sequentially
	for run, runConf := range c.Runs {
		stats := defaultStats(c, run, &runConf)
		startRun(c, run, plat, timeout, stats)

		if run == 0 {
			stats.WriteHeader(csvFile)
		}

		stats.WriteValues(csvFile)
	}

	fmt.Println("Simulation finished")
}

func startRun(c *lib.Config, run int, p platform.Platform,
	t time.Duration,
	stats *monitor.Stats) {
	fmt.Printf("[+] Launching run n°%d\n", run)

	runConf := c.Runs[run]
	// start monitoring first
	mon := monitor.NewMonitor(c.MonitorPort, stats)
	go mon.Listen()
	// then start the platform's run
	doneChan := make(chan bool)
	go func() {
		if err := p.Start(run, &runConf); err != nil {
			panic(err)
		}
		fmt.Printf("[+] platform finished running round %d\n", run)
		go mon.Stop()
		fmt.Printf("[+] Closing down monitor.\n")
		doneChan <- true
	}()
	select {
	case <-doneChan:
		fmt.Printf("[+] Finished.\n")
	case <-time.After(t):
		fmt.Printf("[-] Timed-out.\n")
	}
}

func defaultStats(c *lib.Config, i int, r *lib.RunConfig) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"run":       strconv.Itoa(i),
		"nodes":     strconv.Itoa(r.Nodes),
		"threshold": strconv.Itoa(r.Threshold),
		"network":   c.Network,
	}, nil)
}
