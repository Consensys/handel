package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

var configFile = flag.String("config", "", "config file created for the exp.")

var timeOut = flag.Int("timeOut", 0, "timeout in minutes")
var masterAddr = flag.String("masterAddr", "", "master address")
var network = flag.String("network", "", "network type")
var run = flag.Int("run", 0, "run index")

var resultFile = flag.String("resultFile", "", "result file")
var monitorPort = flag.Int("monitorPort", 0, "monitor port")

var resultsDir string

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	resultsDir = path.Join(currentDir, "results")
}
func main() {
	flag.Parse()
	config := lib.LoadConfig(*configFile)
	runConf := config.Runs[*run]
	nbOfNodes := runConf.Nodes
	//nbOffline := runConf.Failing
	master := lib.NewSyncMaster(*masterAddr, nbOfNodes-runConf.Failing, nbOfNodes)
	fmt.Println("Master: listen on", *masterAddr)

	os.MkdirAll(resultsDir, 0777)
	csvName := filepath.Join(resultsDir, *resultFile)
	//	csvFile, err := os.Create(csvName)
	csvFile, err := os.OpenFile(csvName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	stats := defaultStats(runConf,
		*run,
		*network,
		runConf.Handel.Period,
		config.Simulation,
	)
	mon := monitor.NewMonitor(10000, stats)
	go mon.Listen()

	if strings.Contains(config.Simulation, "libp2p") {
		fmt.Println(" MASTER --->> SYNCING P2P ")
		select {
		case <-master.WaitAll(lib.P2P):
			fmt.Printf("[+] Master full synchronization done.\n")

		case <-time.After(time.Duration(*timeOut) * time.Minute):
			msg := fmt.Sprintf("timeout after %d mn", *timeOut)
			fmt.Println(msg)
		}
		fmt.Println(" MASTER --->> SYNCING P2P DONE ")
	}

	select {
	case <-master.WaitAll(lib.START):
		fmt.Printf("[+] Master full synchronization done.\n")

	case <-time.After(time.Duration(*timeOut) * time.Minute):
		msg := fmt.Sprintf("timeout after %d mn", *timeOut)
		fmt.Println(msg)
	}

	select {
	case <-master.WaitAll(lib.END):
		fmt.Printf("[+] Master - finished synchronization done.\n")
	case <-time.After(time.Duration(25) * time.Second):
		msg := fmt.Sprintf("timeout after %d sec", 25)
		fmt.Println(msg)
	}

	fmt.Println("Writting to", csvName)

	if *run == 0 {
		stats.WriteHeader(csvFile)
	}
	stats.WriteValues(csvFile)
	fmt.Printf("[+] -- MASTER monitor received %d measurements --\n", stats.Received())
	mon.Stop()
}

func defaultStats(runConf lib.RunConfig, run int, network, period, simulation string) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"run":                        strconv.Itoa(run),
		"totalNbOfNodes":             strconv.Itoa(runConf.Nodes),
		"nbOfInstances":              strconv.Itoa(runConf.Processes),
		"threshold":                  strconv.Itoa(runConf.Threshold),
		"failing":                    strconv.Itoa(runConf.Failing),
		"network":                    network,
		"period":                     runConf.Handel.Period,
		"updateCount":                strconv.Itoa(runConf.Handel.UpdateCount),
		"simulation":                 simulation,
		"UnsafeSleepTimeOnSigVerify": strconv.Itoa(runConf.Handel.UnsafeSleepTimeOnSigVerify),
		"NodeCount":                  strconv.Itoa(runConf.Handel.NodeCount),
		"timeout":                    runConf.Handel.Timeout,
	}, nil)
}
