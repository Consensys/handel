package lib

import (
	"errors"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ConsenSys/handel"
	cf "github.com/ConsenSys/handel/bn256/cf"
	golang "github.com/ConsenSys/handel/bn256/go"
	"github.com/ConsenSys/handel/network"
	"github.com/ConsenSys/handel/network/quic"
	"github.com/ConsenSys/handel/network/udp"
	"github.com/ConsenSys/handel/simul/monitor"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var resultsDir string

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	resultsDir = path.Join(currentDir, "results")
	os.MkdirAll(resultsDir, 0777)
}

// Message that will get signed
var Message = []byte("Everything that is beautiful and noble is the product of reason and calculation.")

// Config is read from a TOML encoded file and passed to Platform.Config and
// prepares the platform for specific system-wide configurations.
type Config struct {
	// private fields do not get marshalled
	configPath string
	// which network should we use
	// Valid value: "udp" (default)
	Network string
	// which "curve system" should we use
	// Valid value: "bn256" (default)
	Curve string
	// which encoding should we use on the network
	// valid value: "gob" (default)
	Encoding string
	// which allocator to use when experimenting failing nodes
	// valid value: "round" (default) or "random"
	Allocator string
	// which is the port to send measurements to
	MonitorPort int
	// Debug forwards the debug output if set to != 0
	Debug int
	// which simulation are we running -
	// valid values: "handel" (default) or "p2p/udp" or "p2p/libp2p"
	Simulation string
	// Maximum time to wait for the whole thing to finish
	// string because of ugly format of TOML encoding ---
	MaxTimeout string
	// how many time should we repeat each experiment
	Retrials int
	// to which file should we write the results
	ResultFile string
	// config for each run
	Runs []RunConfig
}

// RunConfig is the config holding parameters for a specific run. A platform can
// start multiple runs sequentially with different parameters each.
type RunConfig struct {
	// How many nodes should we spin for this run
	Nodes int
	// threshold of signatures to wait for
	Threshold int
	// Number of failing nodes
	Failing int
	// Number of processes for this run
	Processes int
	// Handel items configurable  - will be merged with defaults
	Handel *HandelConfig
	// extra for particular information for specific platform for examples
	Extra map[string]string
}

// HandelConfig is a small config that will be converted to handel.Config during
// the simulation
type HandelConfig struct {
	// Period of the periodic update loop
	Period string
	// Number of node do we contact for each periodic update
	UpdateCount int
	// Number of node do we contact when starting level + when finishing level
	// XXX - maybe remove in the futur ! -
	NodeCount int
	// Timeout used to give to the LinearTimeout constructor
	Timeout string
	// UnsafeSleepTimeOnSigVerify
	UnsafeSleepTimeOnSigVerify int
}

// LoadConfig looks up the given file to unmarshal a TOML encoded Config.
func LoadConfig(path string) *Config {
	c := new(Config)
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		panic(err)
	}
	if c.MonitorPort == 0 {
		c.MonitorPort = monitor.DefaultSinkPort
	}
	if c.Simulation == "" {
		c.Simulation = "handel"
	}
	c.configPath = path
	return c
}

// WriteTo writes the config to the specified file path.
func (c *Config) WriteTo(path string) error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}

	enc := toml.NewEncoder(file)
	return enc.Encode(c)
}

// Logger returns the logger set to the right verbosity with timestamp added
func (c *Config) Logger() handel.Logger {
	var logger handel.Logger
	if c.Debug != 0 {
		logger = handel.NewKitLogger(level.AllowDebug())
	} else {
		logger = handel.NewKitLogger(level.AllowInfo())
	}
	//return logger.With("ts", log.DefaultTimestamp)
	return logger.With("ts", log.TimestampFormat(time.Now, time.StampMilli))
}

// MaxNodes returns the maximum number of nodes to test
func (c *Config) MaxNodes() int {
	max := 0
	for _, rc := range c.Runs {
		if max < rc.Nodes-rc.Failing {
			max = rc.Nodes - rc.Failing
		}
	}
	return max
}

// NewNetwork returns the network implementation designated by this config for this
// given identity
func (c *Config) NewNetwork(id handel.Identity) handel.Network {
	if c.Network == "" {
		c.Network = "udp"
	}
	netw, err := c.selectNetwork(id)
	if err != nil {
		panic(err)
	}
	return netw
}

func (c *Config) selectNetwork(id handel.Identity) (handel.Network, error) {
	encoding := c.NewEncoding()
	switch c.Network {
	case "udp":
		return udp.NewNetwork(id.Address(), encoding)
	case "delayed_udp":
		return udp.NewDelayedUDPNetwork(2000* time.Millisecond, id.Address(), encoding)
	case "quic-test-insecure":
		cfg := quic.NewInsecureTestConfig()
		return quic.NewNetwork(id.Address(), encoding, cfg)
	case "quic":
		return nil, errors.New("quic implemented implemented only in test mode")

	default:
		return nil, errors.New("not implemented yet")
	}
}

// NewEncoding returns the corresponding network encoding
func (c *Config) NewEncoding() network.Encoding {
	newEnc := func() network.Encoding {

		if c.Encoding == "" {
			c.Encoding = "gob"
		}
		switch c.Encoding {
		case "gob":
			return network.NewGOBEncoding()
		default:
			panic("not implemented yet")
		}
	}
	encoding := newEnc()
	return network.NewCounterEncoding(encoding)
}

// NewConstructor returns a Constructor that is using the curve denoted by the
// curve field of the config. Valid input so far is "bn256".
func (c *Config) NewConstructor() Constructor {
	if c.Curve == "" {
		c.Curve = "bn256/cf"
	}
	switch c.Curve {
	case "bn256":
		fallthrough
	case "bn256/cf":
		return &SimulConstructor{cf.NewConstructor()}
	case "bn256/go":
		return &SimulConstructor{golang.NewConstructor()}
	default:
		panic("not implemented yet")
	}
}

// NewAllocator returns the allocation determined by the "Allocator" string field
// of the config.
func (c *Config) NewAllocator() Allocator {
	switch c.Allocator {
	case "round":
		return new(RoundRobin)
	case "random":
		return NewRoundRandomOffline()
	default:
		return new(RoundRobin)
	}
}

// GetMaxTimeout returns the global maximum timeout specified in the config
func (c *Config) GetMaxTimeout() time.Duration {
	dd, err := time.ParseDuration(c.MaxTimeout)
	if err != nil {
		panic(err)
	}
	return dd
}

// GetMonitorAddress returns a full IP address composed of the given address
// apprended with the port from the config.
func (c *Config) GetMonitorAddress(ip string) string {
	return net.JoinHostPort(ip, strconv.Itoa(c.MonitorPort))
}

// GetCSVFile returns a name of the CSV file
func (c *Config) GetCSVFile() string {
	csvName := strings.Replace(filepath.Base(c.configPath), ".toml", ".csv", 1)
	return csvName
}

// GetResultsFile returns the path where to write the resulting csv file
func (c *Config) GetResultsFile() string {
	return filepath.Join(resultsDir, c.GetCSVFile())
}

// GetResultsDir returns the directory where results will be written
func (c *Config) GetResultsDir() string {
	return resultsDir
}

// GetBinaryPath returns the binary to compile
func (c *Config) GetBinaryPath() string {
	base := "github.com/ConsenSys/handel/simul/"
	simul := strings.ToLower(c.Simulation)
	if strings.Contains(simul, "p2p") {
		return filepath.Join(base, simul)
	}
	return filepath.Join(base, "node")
}

// GetThreshold returns the threshold to use for this run config - if 0 it
// returns the number of nodes
func (r *RunConfig) GetThreshold() int {
	if r.Threshold == 0 {
		return r.Nodes
	}
	return r.Threshold
}

// GetHandelConfig returns the config to pass down to handel instances
// Returns the default if not set
func (r *RunConfig) GetHandelConfig() *handel.Config {
	ch := &handel.Config{}
	if r.Handel == nil {
		ch = handel.DefaultConfig(r.Nodes)
		ch.Contributions = r.Threshold
	}
	period, err := time.ParseDuration(r.Handel.Period)
	if err != nil {
		panic(err)
	}
	ch.UpdatePeriod = period
	ch.UpdateCount = r.Handel.UpdateCount
	ch.NodeCount = r.Handel.NodeCount
	ch.Contributions = r.GetThreshold()
	ch.UnsafeSleepTimeOnSigVerify = r.Handel.UnsafeSleepTimeOnSigVerify

	dd, err := time.ParseDuration(r.Handel.Timeout)
	if err == nil {
		ch.NewTimeoutStrategy = handel.LinearTimeoutConstructor(dd)
	}
	return ch
}

// Duration is an alias for time.Duration
type Duration time.Duration

// UnmarshalText implements the TextUnmarshaler interface
func (d *Duration) UnmarshalText(text []byte) error {
	dd, err := time.ParseDuration(string(text))
	if err == nil {
		*d = Duration(dd)
	}
	return err
}

// MarshalText implements the TextMarshaler interface
func (d *Duration) MarshalText() ([]byte, error) {
	str := time.Duration(*d).String()
	return []byte(str), nil
}

// Divmod returns the integer results and remainder of the division
func Divmod(numerator, denominator int) (quotient, remainder int) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder = numerator % denominator
	return
}
