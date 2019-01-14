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
	n := len(slaveAddrs) * 2 // 2 nodes per instances
	master := NewSyncMaster(masterAddr, n, n)
	defer master.Stop()

	var slaves = make([]*SyncSlave, len(slaveAddrs))
	doneSlave := make(chan bool, len(slaveAddrs))
	for i, addr := range slaveAddrs {
		slaves[i] = NewSyncSlave(addr, masterAddr, []int{i * 2, i*2 + 1})
		defer slaves[i].Stop()
	}

	tryWait := func(m *SyncMaster, slaves []*SyncSlave) {
		for i := range slaves {
			go func(j int) {
				fmt.Println("waiting start")
				doneSlave <- <-slaves[j].WaitMaster()
			}(i)
		}
		var masterDone bool
		var slavesDone int

		for {
			select {
			case <-master.WaitAll():
				masterDone = true
			case <-doneSlave:
				slavesDone++
			case <-time.After(1000 * time.Millisecond):
				panic("aie aie aie")
			}
			if masterDone && slavesDone == len(slaveAddrs) {
				return
			}
		}
	}
	tryWait(master, slaves)

	master.Reset()
	for i := range slaves {
		slaves[i].Reset()
	}

	tryWait(master, slaves)
}
