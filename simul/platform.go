// Package simul contains interface and implementation to run a Handel node
// on multiple platforms. Such implementations include Localhost (run your
// test locally).
package simul

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
	Configure(*Config) error
	// Makes sure that there is no part of the application still running
	Cleanup() error
	// Start runs an experiment from this run config. Any implementations MUST
	// perform this experiment a number of times as defined in the Config struct
	// field Retrials. It must be a blocking call that returns only when the
	// experiment is finished.
	Start(*RunConfig) error
}

var deterlab = "deterlab"
var localhost = "localhost"
var mininet = "mininet"

// NewPlatform returns the appropriate platform
// [deterlab,localhost]
func NewPlatform(t string) Platform {
	var p Platform
	switch t {
	case localhost:
		//p = &Localhost{}
	default:
		panic("no platform of this name " + t)
	}
	return p
}
