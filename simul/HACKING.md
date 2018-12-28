# Simulation

## High-Level Overview 

A simulation runs like the following:
1. Read the config file with all the different experiments to try
2. For each run (one experiment):
    + Generate all key pairs necessary in a CSV file
    + Run the binaries
    + Synchronize them: make sure they are all up before starting the protocol
    + Wait the binaries exit
    + Loop to next run

## Main Interfaces

### Config

Config is the main files through which operators can write down experiment
parameters. 
```go
type Config struct {
	// which network should we use
	// Valid value: "udp" (default)
	Network string
	// which "curve system" should we use
	// Valid value: "bn256" (default)
	Curve string
	// which encoding should we use on the network
	// valid value: "gob" (default)
	Encoding string
	// which is the port to send measurements to
	MonitorPort int
	// Debug forwards the debug output if set to != 0
	Debug int
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

type RunConfig struct {
	// How many nodes should we spin for this run
	Nodes int
	// threshold of signatures to wait for
	Threshold int
	// extra for particular information for specific platform for examples
	Extra interface{}
	// XXX NOT USED YET
	//Failing   int
}
```

As you can see, a `Config` holds a list of `RunConfig`: one for each experiment
to perform. 
An operator needs to write down its config in a TOML format. Not all fields are
necessary to set, and an short example config is given in
`simul/config_example.toml`.

### Platform

A Platform is a generic interface to start a given run. 
```go
type Platform interface {
	// + the initial configuration of all structures needed for the platform
	// + building the binaries if needed and deploying them if needed
	Configure(*lib.Config) error
	// Makes sure that there is no part of the application still running
	Cleanup() error
	// Start runs an experiment from this run config, which is at the given
	// index in the Config. 
	Start(idx int, rc *lib.RunConfig) error
}
```

So far only a `Localhost` platform has been implemented. It compiles locally,
and spawns locally multiple binaries.

### Measurement

To collect results from each node (or a subset), one use the `monitor/` package.
In one sentence, one instantiate a `Sink` that collects all measurements sent
via a TCP connection. Later on, this `Sink` can generate CSV file containing the
min,max, avg, std_dev,etc for each measurements.

This package can handle different measures: time measures,  "counter-based", or
simple values, etc. Each measure must fullfill the `Measure` interface:
```go
type Measure interface {
	// Record must be called when you want to send the value
	// over the monitor listening.
	// Implementation of this interface must RESET the value to `0` at the end
	// of Record(). `0` means the initial value / meaning this measure had when
	// created.
	// Example: TimeMeasure.Record() will reset the time to `time.Now()`
	//          CounterMeasure.Record() will  reset the counter of the bytes
	//          read / written to 0.
    Record()
}
```
For example, in Handel, we would like to measure the number of packets sent. We
need the network to implement an interface called `Counter`:
```go
type Counter interface {
	Values() map[string]float64
}
```

Then one can create a `Measure` with `NewCounterMeasure()`.
