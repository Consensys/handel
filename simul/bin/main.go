// Package bin holds the logic of a single Handel node for the simulation
package bin

import (
	"flag"

	"github.com/ConsenSys/handel/simul"
)

var configFile = flag.String("config", "", "config file created for the exp.")
var keyPath = flag.String("key", "", "key-pair file path")
var registryFile = flag.String("registry-file", "", "registry file based - array registry")

// XXX maybe try with a database-backed registry if loading file in memory is
// too much when overloading

func main() {
	flag.Parse()
	config := simul.LoadConfig(*configFile)
}
