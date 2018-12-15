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
var pemFile = flag.String("pemFile", "", "location of the .pem file for EC2 ssh")
var regions = flag.String("regions", "us-west-2", "list of AWS regions")

func main() {
	flag.Parse()

	parameters := make(map[string]string)
	parameters["pemFile"] = *pemFile
	parameters["regions"] = *regions
	c := lib.LoadConfig(*configFlag)
	plat := platform.NewPlatform(*platformFlag, parameters)
	if err := plat.Configure(c); err != nil {
		panic(err)
	}

	plat.Cleanup()

	timeout := *runTimeout * time.Duration(c.Retrials)
	for i, rc := range c.Runs {
		fmt.Printf("[+] Launching run n°%d\n", i)
		doneChan := make(chan bool)
		go func(j int, r *lib.RunConfig) {
			if err := plat.Start(j, r); err != nil {
				panic(err)
			}
			doneChan <- true
		}(i, &rc)
		select {
		case <-doneChan:
			fmt.Printf("Finished.\n")
		case <-time.After(timeout):
			fmt.Printf("Timed-out.\n")
		}
	}

	fmt.Println("Simulation finished")
}
