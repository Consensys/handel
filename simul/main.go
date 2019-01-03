// This package can launches a Handel simulation. It works the following way:
// 1. Read the config TOML file
// 2. Construct the right platform from the flag
// 3. Gives the Config to the Platform
// 4. Run the platform's Run
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/platform"
)

var configFlag = flag.String("config", "", "TOML encoded config file")
var platformFlag = flag.String("platform", "", "name of the platform to run on")
var runTimeout = flag.Duration("run-timeout", 2*time.Minute, "timeout of a given run")

var awsConfigPath = flag.String("awsConfig", "", "TOML encoded config file AWS specyfic config")

func main() {
	flag.Parse()

	c := lib.LoadConfig(*configFlag)
	plat := platform.NewPlatform(*platformFlag, *awsConfigPath)
	if err := plat.Configure(c); err != nil {
		panic(err)
	}

	// preparation phase
	defer plat.Cleanup()

	timeout := *runTimeout * time.Duration(c.Retrials)

	// running rounds sequentially
	for run := range c.Runs {
		startRun(c, run, plat, timeout)
	}

	fmt.Println("Simulation finished")
}

func startRun(c *lib.Config, run int, p platform.Platform, t time.Duration) {
	fmt.Printf("[+] Launching run n°%d\n", run)
	runConf := c.Runs[run]
	// then start the platform's run
	doneChan := make(chan bool)
	go func() {
		if err := p.Start(run, &runConf); err != nil {
			panic(err)
		}
		fmt.Printf("[+] platform finished running round %d\n", run)
		doneChan <- true
	}()
	select {
	case <-doneChan:
		fmt.Printf("[+] Finished.\n")
	case <-time.After(t):
		fmt.Printf("[-] Timed-out.\n")
	}
}
