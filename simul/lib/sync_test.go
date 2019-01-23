package lib

import (
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

	tryWait := func(stateID int, m *SyncMaster, slaves []*SyncSlave) {
		for i := range slaves {
			go func(j int) {
				//slaves[j].SignalAll(stateID)
				for _, id := range slaves[j].ids {
					slaves[j].Signal(stateID, id)
				}
				doneSlave <- <-slaves[j].WaitMaster(stateID)
			}(i)
		}
		var masterDone bool
		var slavesDone int

		for {
			select {
			case <-master.WaitAll(stateID):
				masterDone = true
			case <-doneSlave:
				slavesDone++
			case <-time.After(2000 * time.Millisecond):
				panic("aie aie aie")
			}
			if masterDone && slavesDone == len(slaveAddrs) {
				return
			}
		}
	}
	tryWait(START, master, slaves)
	time.Sleep(50 * time.Millisecond)
	tryWait(END, master, slaves)
}
