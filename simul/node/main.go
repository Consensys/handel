// Package main holds the logic of a single Handel node for the simulation
package main

import (
	"flag"
	"fmt"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

var beaconBytes = []byte{0x01, 0x02, 0x03}

// BeaconTimeout represents how much time do we wait to receive the beacon
const BeaconTimeout = 2 * time.Minute

var configFile = flag.String("config", "", "config file created for the exp.")
var registryFile = flag.String("registry", "", "registry file based - array registry")
var id = flag.Int("id", -1, "peer id")
var run = flag.Int("run", -1, "which RunConfig should we run")
var master = flag.String("master", "", "master address to synchronize")
var syncAddr = flag.String("sync", "", "address to listen for master START")
var monitorAddr = flag.String("monitor", "", "address to send measurements")

var isMonitoring bool

func main() {
	flag.Parse()
	//
	// SETUP PHASE
	//
	if *monitorAddr != "" {
		isMonitoring = true
		if err := monitor.ConnectSink(*monitorAddr); err != nil {
			panic(err)
		}
		defer monitor.EndAndCleanup()
	}
	// first load the measurement unit if needed
	// load all needed structures
	// XXX maybe try with a database-backed registry if loading file in memory is
	// too much when overloading
	config := lib.LoadConfig(*configFile)
	runConf := config.Runs[*run]

	cons := config.NewConstructor()
	parser := lib.NewCSVParser()
	registry, node, err := lib.ReadAll(*registryFile, *id, parser, cons)
	network := config.NewNetwork(node.Identity)

	// make the signature
	signature, err := node.Sign(lib.Message, nil)
	if err != nil {
		panic(err)
	}
	// Setup report handel
	handel := h.NewHandel(network, registry, node.Identity, cons.Handel(), lib.Message, signature)
	reporter := h.NewReportHandel(handel)

	// Sync with master - wait for the START signal
	syncer := lib.NewSyncSlave(*syncAddr, *master, *id)
	select {
	case <-syncer.WaitMaster():
		now := time.Now()
		formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond())

		fmt.Printf("\n%s [+] %s synced - starting\n", formatted, node.Identity.Address())
	case <-time.After(BeaconTimeout):
		panic("Haven't received beacon in time!")
	}

	// Start handel and run a timeout on the whole thing
	signatureGen := monitor.NewTimeMeasure("sigen")
	go reporter.Start()
	out := make(chan bool, 1)
	go func() {
		<-time.After(config.GetMaxTimeout())
		out <- true
	}()

	// Wait for final signatures !
	enough := false
	var sig h.MultiSignature
	for !enough {
		select {
		case sig = <-reporter.FinalSignatures():
			if sig.BitSet.Cardinality() >= runConf.Threshold {
				enough = true
				break
			}
		case <-out:
			panic("max timeout")
		}
	}
	signatureGen.Record()
	fmt.Println("reached good enough multi-signature!")

	if err := h.VerifyMultiSignature(lib.Message, &sig, registry, cons.Handel()); err != nil {
		panic("signature invalid !!")
	}

	fmt.Println("signature valid & finished- sending state to sync master")

	// Sync with master - wait to close our node
	syncer.Reset()
	select {
	case <-syncer.WaitMaster():
		now := time.Now()
		formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond())

		fmt.Printf("\n%s [+] %s synced - closing shop\n", formatted, node.Identity.Address())
	case <-time.After(BeaconTimeout):
		panic("Haven't received beacon in time!")
	}
}
