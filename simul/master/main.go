package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
)

var nbOfNodes = flag.Int("nbOfNodes", 0, "number of slave nodes")
var masterAddr = flag.String("masterAddr", "", "master address")

func main() {
	flag.Parse()
	master := lib.NewSyncMaster(*masterAddr, *nbOfNodes)
	fmt.Println("Master: listen on", *masterAddr)

	select {
	case <-master.WaitAll():
		fmt.Printf("[+] Master full synchronization done.\n")
		master.Reset()

	case <-time.After(3 * time.Minute):
		fmt.Println("timeout after 2 mn")
		panic("timeout after 2 mn")
	}

	// 5. Wait all finished - then tell them to quit
	select {
	case <-master.WaitAll():
		fmt.Printf("[+] Master - finished synchronization done.\n")
	case <-time.After(3 * time.Minute):
		fmt.Println("timeout after 3 mn")

		panic(fmt.Sprintf("timeout after 3 mn"))
	}

}
