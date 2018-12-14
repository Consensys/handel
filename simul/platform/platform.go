// Package platform contains interface and implementation to run a Handel node
// on multiple platforms. Such implementations include Localhost (run your
// test locally).
package platform

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ConsenSys/handel/simul/lib"
)

// The Life of a simulation:
//
// 1. Configure
//     * read configuration
//     * compile eventual files
// 2. Build
//     * builds all files
//     * eventually for different platforms
// 3. Cleanup
//     * send killall to applications
// 4. Deploy
//     * make sure the environment is up and running
//     * copy files
// 5. Start
//     * start all logservers
//     * start all nodes
//     * start all clients
// 6. Wait
//     * wait for the applications to finish

// Platform interface that has to be implemented to add another simulation-
// platform.
type Platform interface {
	// + the initial configuration of all structures needed for the platform
	// + building the binaries if needed and deploying them if needed
	Configure(*lib.Config) error
	// Makes sure that there is no part of the application still running
	Cleanup() error
	// Start runs an experiment from this run config, which is at the given
	// index in the Config. Both informations are redundant however it avoids
	// one useless step for each platform. Any implementations MUST perform this
	// experiment a number of times as defined in the Config struct field
	// Retrials. It must be a blocking call that returns only when the
	// experiment is finished.
	Start(idx int, rc *lib.RunConfig) error
}

var localhost = "localhost"

// NewPlatform returns the appropriate platform [deterlab,localhost]
// and setups the Cleanup call in case of a signal interruption
func NewPlatform(t string) Platform {
	var p Platform
	switch t {
	case localhost:
		p = NewLocalhost()
	default:
		panic("no platform of this name " + t)
	}
	catchSIGINT(p)
	return p
}

func catchSIGINT(p Platform) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		if err := p.Cleanup(); err == nil {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}()
}
