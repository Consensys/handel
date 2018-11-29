// Package simul can launches a Handel simulation. It works the following way:
// 1. Read the config TOML file
// 2. Construct the right platform from the flag
// 3. Gives the Config to the Platform
// 4. Run the platform's Run
package simul

import (
	"flag"
	"fmt"
	"time"
)

var configFlag = flag.String("config", "", "TOML encoded config file")
var platformFlag = flag.String("platform", "", "name of the platform to run on")
var runTimeout = flag.Duration("run-timeout", time.Minute, "timeout of a given run")

func main() {
	flag.Parse()
	c := LoadConfig(*configFlag)

	plat := NewPlatform(*platformFlag)
	if err := plat.Configure(c); err != nil {
		panic(err)
	}

	plat.Cleanup()

	timeout := *runTimeout * time.Duration(c.Retrials)
	for i, rc := range c.Runs {
		fmt.Printf("[+] starting run %d ...", i)
		doneChan := make(chan bool)
		go func(r *RunConfig) {
			if err := plat.Start(r); err != nil {
				panic(err)
			}
			doneChan <- true
		}(&rc)
		select {
		case <-doneChan:
			fmt.Printf("Finished.\n")
		case <-time.After(timeout):
			fmt.Printf("Timed-out.\n")
		}
	}

	fmt.Println("Simulation finished")
}
