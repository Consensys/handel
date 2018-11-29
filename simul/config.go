package simul

import "github.com/BurntSushi/toml"

// Message that will get signed
var Message = []byte("Everything that is beautiful and noble is the product of reason and calculation.")

// Config is read from a TOML encoded file and passed to Platform.Config and
// prepares the platform for specific system-wide configurations.
type Config struct {
	// which network should we use
	// Possible values are "udp" "tcp" "libp2p-tcp" "libp2p-udp" ...
	Network string
	// which "curve system" should we use
	// Possible values are "bn256" "bls12-381" ...
	Curve string
	// which is the port to send measurements to
	MonitorPort int
	// Debug forwards the debug output if set to != 0
	Debug int
	// how many time should we repeat each experiment
	Retrials int
	// to which file should we write the results
	ResultFile string
	// config for each run
	Runs []RunConfig
}

// MaxNodes returns the maximum number of nodes to test
func (c *Config) MaxNodes() int {
	max := 0
	for _, rc := range c.Runs {
		if max < rc.Nodes {
			max = rc.Nodes
		}
	}
	return max
}

// RunConfig is the config holding parameters for a specific run. A platform can
// start multiple runs sequentially with different parameters each.
type RunConfig struct {
	Nodes int
	// extra for particular information for specific platform for examples
	Extra interface{}
	// XXX NOT USED YET
	//Threshold int
	//Failing   int
}

// LoadConfig looks up the given file to unmarshal a TOML encoded Config.
func LoadConfig(path string) *Config {
	c := new(Config)
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		panic(err)
	}
	return c
}
