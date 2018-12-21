package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
)

var nbOfNodes = flag.Int("nbOfNodes", 0, "number of slave nodes")
var timeOut = flag.Int("timeOut", 0, "timeout in minutes")
var masterAddr = flag.String("masterAddr", "", "master address")

func main() {
	flag.Parse()
	master := lib.NewSyncMaster(*masterAddr, *nbOfNodes)
	fmt.Println("Master: listen on", *masterAddr)

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
}
