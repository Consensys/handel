package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/p2p"
)

func main() {

	flag.Parse()

	maker := p2p.AdaptorFunc(MakeP2P)
	maker = p2p.WithPostFunc(maker, func(r handel.Registry, nodes []p2p.Node) {
		config := lib.LoadConfig(*p2p.ConfigFile)
		fmt.Println(" libp2pBINARY --->> SYNCING P2P on", *p2p.SyncAddr)
		syncer := lib.NewSyncSlave(*p2p.SyncAddr, *p2p.Master, p2p.Ids)
		syncer.SignalAll(lib.P2P)
		select {
		case <-syncer.WaitMaster(lib.P2P):
			now := time.Now()
			formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
				now.Minute(),
				now.Second(),
				now.Nanosecond())

			fmt.Printf("\n%s [+] %s synced - starting\n", formatted, p2p.Ids.String())
		case <-time.After(config.GetMaxTimeout()):
			panic("Haven't received beacon in time!")
		}
		fmt.Println(" libp2pBINARY --->> SYNCING P2P DONE")
		syncer.Stop()
	})
	maker = p2p.WithConnector(maker)
	maker = p2p.WithPostFunc(maker, func(r handel.Registry, nodes []p2p.Node) {
		var wg sync.WaitGroup
		for _, n := range nodes {
			wg.Add(1)
			go func(n *P2PNode) {
				n.WaitAllSetup()
				wg.Done()
			}(n.(*P2PNode))
		}
		wg.Wait()
	})

	p2p.Run(maker)
}
