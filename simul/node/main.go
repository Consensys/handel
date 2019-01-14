// Package main holds the logic of a single Handel node for the simulation
package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

// BeaconTimeout represents how much time do we wait to receive the beacon
const BeaconTimeout = 2 * time.Minute

var configFile = flag.String("config", "", "config file created for the exp.")
var registryFile = flag.String("registry", "", "registry file based - array registry")
var ids arrayFlags

var run = flag.Int("run", -1, "which RunConfig should we run")
var master = flag.String("master", "", "master address to synchronize")
var syncAddr = flag.String("sync", "", "address to listen for master START")
var monitorAddr = flag.String("monitor", "", "address to send measurements")

func init() {
	flag.Var(&ids, "id", "ID to run on this node - can specify multiple -id flags")
}

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
	nodeList, err := lib.ReadAll(*registryFile, parser, cons)
	if err != nil {
		panic(err)
	}
	registry := nodeList.Registry()

	// instantiate handel for all specified ids in the flags
	var handels []*h.ReportHandel
	for _, id := range ids {
		fmt.Println(nodeList)
		node := nodeList.Node(id)
		network := config.NewNetwork(node.Identity)

		// make the signature
		signature, err := node.Sign(lib.Message, nil)
		if err != nil {
			panic(err)
		}
		// Setup report handel
		handel := h.NewHandel(network, registry, node.Identity, cons.Handel(), lib.Message, signature)
		reporter := h.NewReportHandel(handel)
		handels = append(handels, reporter)
	}

	// Sync with master - wait for the START signal
	syncer := lib.NewSyncSlave(*syncAddr, *master, ids)
	select {
	case <-syncer.WaitMaster():
		now := time.Now()
		formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond())

		fmt.Printf("\n%s [+] %s synced - starting\n", formatted, ids.String())
	case <-time.After(BeaconTimeout):
		panic("Haven't received beacon in time!")
	}

	// Start all handels and run a timeout on the signature generation time
	var wg sync.WaitGroup
	for i := range handels {
		wg.Add(1)
		go func(j int) {
			handel := handels[j]
			id := ids[j]
			signatureGen := monitor.NewTimeMeasure("sigen")
			netMeasure := monitor.NewCounterMeasure("net", handel.Network())
			storeMeasure := monitor.NewCounterMeasure("store", handel.Store())
			go handel.Start()
			// Wait for final signatures !
			enough := false
			var sig h.MultiSignature
			for !enough {
				select {
				case sig = <-handel.FinalSignatures():
					if sig.BitSet.Cardinality() >= runConf.Threshold {
						enough = true
						wg.Done()
						fmt.Printf(" --- NODE  %d FINISHED ---\n", id)
						break
					}
				case <-time.After(config.GetMaxTimeout()):
					panic("max timeout")
				}
			}
			signatureGen.Record()
			netMeasure.Record()
			storeMeasure.Record()
			fmt.Println("reached good enough multi-signature!")

			if err := h.VerifyMultiSignature(lib.Message, &sig, registry, cons.Handel()); err != nil {
				panic("signature invalid !!")
			}
		}(i)
	}
	wg.Wait()
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

		fmt.Printf("\n%s [+] %s synced - closing shop\n", formatted, ids.String())
	case <-time.After(BeaconTimeout):
		panic("Haven't received beacon in time!")
	}
}

type arrayFlags []int

func (i *arrayFlags) String() string {
	var a = make([]string, len(*i))
	for i, v := range *i {
		a[i] = strconv.Itoa(v)
	}
	return strings.Join(a, "-")
}

func (i *arrayFlags) Set(value string) error {
	newID, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*i = append(*i, newID)
	return nil
}
