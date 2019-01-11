package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
	golog "github.com/ipfs/go-log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	gologging "github.com/whyrusleeping/go-logging"
)

// MaxCount represents the number of outgoing connections a gossip node should
// make
const MaxCount = 10

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

	if true {
		golog.SetAllLoggers(gologging.INFO)
	}

	flag.Parse()
	//
	// SETUP PHASE
	//

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	registry, aggregators := ReadRegistry(ctx, *registryFile, parser, cons, ids,
		extractRouter(&runConf))
	list := registry.(*P2PRegistry)
	// connect the nodes - create the overlay
	connector, count := extractConnector(&runConf)
	for _, agg := range aggregators {
		err := connector.Connect(agg.P2PNode, []*P2PIdentity(*list), count)
		if err != nil {
			fmt.Println("err : ", err)
			panic(err)
		}
	}

	fmt.Println(" overlay network  - connections - setup ")
	time.Sleep(10 * time.Second)
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

	// Start all aggregators and run a timeout on the signature generation time
	var wg sync.WaitGroup
	var report = make(chan int, len(aggregators))
	for i := range aggregators {
		wg.Add(1)
		go func(j int) {
			agg := aggregators[j]
			id := agg.handelID
			signatureGen := monitor.NewTimeMeasure("sigen")
			go agg.Start()
			// Wait for final signatures !
			enough := false
			var sig *h.MultiSignature
			for !enough {
				select {
				case sig = <-agg.FinalMultiSignature():
					if sig.BitSet.Cardinality() >= runConf.Threshold {
						enough = true
						report <- int(id)
						wg.Done()
						break
					}
				case <-time.After(config.GetMaxTimeout()):
					panic("max timeout")
				}
			}
			signatureGen.Record()
			fmt.Println("reached good enough multi-signature!")

			if err := h.VerifyMultiSignature(lib.Message, sig, registry, cons.Handel()); err != nil {
				panic("signature invalid !!")
			}
		}(i)
	}

	go func() {
		total := len(aggregators)
		curr := 1
		for i := range report {
			fmt.Printf(" --- NODE  %d  - %d/%d FINISHED ---\n", i, curr, total)
			curr++
		}
	}()
	wg.Wait()
	close(report)
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

// ReadRegistry extracts a list of P2PIdentity and the relevant Aggregators from the
// registry directly - alleviating the need for keeping a second list.
func ReadRegistry(ctx context.Context, uri string, parser lib.NodeParser, c lib.Constructor, ids []int, r NewRouter) (h.Registry, []*Aggregator) {
	records, err := parser.Read(uri)
	if err != nil {
		panic(err)
	}
	total := len(records)

	pubsub.GossipSubHistoryLength = total
	pubsub.GossipSubHistoryGossip = total
	pubsub.GossipSubHeartbeatInterval = 500 * time.Millisecond

	var aggregators = make([]*Aggregator, 0, len(ids))
	var registry = P2PRegistry(make([]*P2PIdentity, total))
	for _, rec := range records {
		node, err := rec.ToNode(c)
		if err != nil {
			panic(err)
		}
		id := int(node.ID())
		registry[id], err = NewP2PIdentity(node.Identity)
		if err != nil {
			panic(err)
		}

		if isIncluded(ids, id) {
			p2pNode, err := NewP2PNode(ctx, node, r)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
			agg := NewAggregator(p2pNode, &registry, c.Handel(), total)
			aggregators = append(aggregators, agg)
		}
	}
	return &registry, aggregators
}

func isIncluded(arr []int, v int) bool {
	for _, a := range arr {
		if v == a {
			return true
		}
	}
	return false
}

func extractConnector(r *lib.RunConfig) (Connector, int) {
	c, exists := r.Extra["Connector"]
	if !exists {
		c = "neighbor"
	}
	countStr, exists := r.Extra["Count"]
	count := MaxCount
	if exists {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			panic(err)
		}
	}
	var con Connector
	switch strings.ToLower(c) {
	case "neighbor":
		con = NewNeighborConnector()
		fmt.Println(" selecting NEIGHBOR connector with ", count)
	case "random":
		con = NewRandomConnector()
		fmt.Println(" selecting RANDOM connector with ", count)
	}
	return con, count

}

func extractRouter(r *lib.RunConfig) NewRouter {
	str, exists := r.Extra["Router"]
	if !exists {
		str = "flood"
	}

	var n NewRouter
	switch strings.ToLower(str) {
	case "flood":
		fmt.Println("using flood router")
		n = pubsub.NewFloodSub
	case "gossip":
		n = pubsub.NewGossipSub
	default:
		n = pubsub.NewGossipSub
	}
	return n
}
