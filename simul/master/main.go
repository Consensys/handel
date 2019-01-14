package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
)

var nbOfNodes = flag.Int("nbOfNodes", 0, "total number of slave nodes")
var nbOffline = flag.Int("nbOffline", 0, "number of offline nodes")
var nbOfInstances = flag.Int("nbOfInstances", 0, "number of slave instances")

var timeOut = flag.Int("timeOut", 0, "timeout in minutes")
var masterAddr = flag.String("masterAddr", "", "master address")
var network = flag.String("network", "", "network type")
var run = flag.Int("run", 0, "run index")
var threshold = flag.Int("threshold", 0, "min threshold of contributions")
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
	active := *nbOfNodes - *nbOffline
	master := lib.NewSyncMaster(*masterAddr, active, *nbOfNodes)
	fmt.Println("Master: listen on", *masterAddr)

	os.MkdirAll(resultsDir, 0777)
	csvName := filepath.Join(resultsDir, *resultFile)
	//	csvFile, err := os.Create(csvName)
	csvFile, err := os.OpenFile(csvName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	stats := defaultStats(*run, *nbOfNodes, *threshold, *nbOfInstances, *network)
	mon := monitor.NewMonitor(10000, stats)
	go mon.Listen()

	select {
	case <-master.WaitAll():
		fmt.Printf("[+] Master full synchronization done.\n")
		master.Reset()

	case <-time.After(time.Duration(*timeOut) * time.Minute):
		msg := fmt.Sprintf("timeout after %d mn", *timeOut)
		fmt.Println(msg)
		panic(fmt.Sprintf("timeout after %d mn", *timeOut))
	}

	select {
	case <-master.WaitAll():
		fmt.Printf("[+] Master - finished synchronization done.\n")
	case <-time.After(time.Duration(*timeOut) * time.Minute):
		msg := fmt.Sprintf("timeout after %d mn", *timeOut)
		fmt.Println(msg)
		panic(msg)
	}

	if *run == 0 {
		stats.WriteHeader(csvFile)
	}
	stats.WriteValues(csvFile)
	mon.Stop()
}

func defaultStats(run, nodes, threshold, nbOfInstances int, network string) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"run":            strconv.Itoa(run),
		"totalNbOfNodes": strconv.Itoa(nodes),
		"nbOfInstances":  strconv.Itoa(nbOfInstances),
		"threshold":      strconv.Itoa(threshold),
		"network":        network,
	}, nil)
}
