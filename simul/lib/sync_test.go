package lib

import (
	"fmt"
	"testing"
	"time"
)

func TestSyncer(t *testing.T) {
	masterAddr := "127.0.0.1:3000"
	slaveAddrs := []string{
		"127.0.0.1:3001",
		"127.0.0.1:3002",
		"127.0.0.1:3003",
	}
	n := len(slaveAddrs)
	master := NewSyncMaster(masterAddr, 3)

	var slaves = make([]*SyncSlave, len(slaveAddrs))
	doneSlave := make(chan bool, len(slaveAddrs))
	for i, addr := range slaveAddrs {
		slaves[i] = NewSyncSlave(addr, masterAddr)
		go func(j int) {
			fmt.Println("waiting start")
			doneSlave <- <-slaves[j].WaitStart()
		}(i)
	}

	var masterDone bool
	var slavesDone int

	for {
		select {
		case <-master.WaitAllSetup():
			masterDone = true
		case <-doneSlave:
			slavesDone++
		case <-time.After(1000 * time.Millisecond):
			panic("aie aie aie")
		}
		if masterDone && slavesDone == n {
			return
		}
	}
}
