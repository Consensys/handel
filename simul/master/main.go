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
	"github.com/ConsenSys/handel/simul/platform"
)

var nbOfNodes = flag.Int("nbOfNodes", 0, "number of slave nodes")
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
	master := lib.NewSyncMaster(*masterAddr, *nbOfNodes)
	fmt.Println("Master: listen on", *masterAddr)

	os.MkdirAll(resultsDir, 0777)
	csvName := filepath.Join(resultsDir, *resultFile)
	csvFile, err := os.Create(csvName)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	stats := platform.DefaultStats(*run, *nbOfNodes, *threshold, *network)
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

	stats.WriteHeader(csvFile)
	stats.WriteValues(csvFile)
	mon.Stop()
}

func defaultStats(nbOfNodes int) *monitor.Stats {
	return monitor.NewStats(map[string]string{
		"nodes": strconv.Itoa(nbOfNodes),
	}, nil)
}
