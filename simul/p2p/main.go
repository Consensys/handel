package p2p

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/monitor"
	golog "github.com/ipfs/go-log"
	gologging "github.com/whyrusleeping/go-logging"
)

// CtxKey is the type inserted at key value in context
type CtxKey string

// MaxCount represents the number of outgoing connections a gossip node should
// make
const MaxCount = 10

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

// Run starts the simulation
func Run(a Adaptor) {

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
	ctx = context.WithValue(ctx, CtxKey("Constructor"), cons) // for libp2p
	parser := lib.NewCSVParser()
	// read CSV records
	records, err := parser.Read(*registryFile)
	requireNil(err)
	// transform into lib.Node
	libNodes, err := toLibNodes(cons, records)
	registry, p2pNodes := a.Make(ctx, libNodes, ids, runConf.GetThreshold(), runConf.Extra)
	aggregators := MakeAggregators(ctx, cons, p2pNodes, registry, runConf.GetThreshold(), runConf.Extra)

	// Sync with master - wait for the START signal
	syncer := lib.NewSyncSlave(*syncAddr, *master, ids)
	syncer.SignalAll(lib.START)
	select {
	case <-syncer.WaitMaster(lib.START):
		now := time.Now()
		formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond())

		fmt.Printf("\n%s [+] %s synced - starting\n", formatted, ids.String())
	case <-time.After(config.GetMaxTimeout()):
		panic("Haven't received beacon in time!")
	}

	// Start all aggregators and run a timeout on the signature generation time
	var wg sync.WaitGroup
	var report = make(chan int, len(aggregators))
	for i := range aggregators {
		wg.Add(1)
		go func(j int) {
			agg := aggregators[j]
			id := agg.Identity().ID()
			//fmt.Println(" --- LAUNCHING agg j = ", j, " vs pk = ", agg.Identity().PublicKey().String())
			signatureGen := monitor.NewTimeMeasure("sigen")
			go agg.Start()
			// Wait for final signatures !
			enough := false
			var sig *h.MultiSignature
			for !enough {
				select {
				case sig = <-agg.FinalMultiSignature():
					if sig.BitSet.Cardinality() >= runConf.Threshold {
						//fmt.Printf(" --- NODE %d outputted signature of %d / %d contributions\n", id, sig.BitSet.Cardinality(), runConf.Threshold)
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
			if err := h.VerifyMultiSignature(lib.Message, sig, registry, cons.Handel()); err != nil {
				panic("signature invalid !!")
			}
			syncer.Signal(lib.END, int(id))
		}(i)
	}

	go func() {
		total := len(aggregators)
		curr := 1
		for i := range report {
			fmt.Printf(" --- NODE  %d FINISHED - in process: %d/%d ---\n", i, curr, total)
			curr++
		}
	}()
	wg.Wait()
	close(report)
	fmt.Println("signature valid & finished- sending state to sync master")

	// Sync with master - wait to close our node
	select {
	case <-syncer.WaitMaster(lib.END):
		now := time.Now()
		formatted := fmt.Sprintf("%02d:%02d:%02d:%03d", now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond())

		fmt.Printf("\n%s [+] %s synced - closing shop\n", formatted, ids.String())
	case <-time.After(config.GetMaxTimeout()):
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

// MakeAggregators returns
func MakeAggregators(ctx context.Context, c lib.Constructor, nodes []Node, reg handel.Registry, threshold int, opts Opts) []*Aggregator {
	resendPeriod := extractResendPeriod(opts)
	aggAndVerify := extractAggTechnique(opts)
	var aggs = make([]*Aggregator, 0, len(nodes))
	for _, node := range nodes {
		//i := int(node.Identity().ID())
		sig, err := node.SecretKey().Sign(lib.Message, nil)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		agg := NewAggregator(ctx, node, reg, c.Handel(), sig, threshold, resendPeriod, aggAndVerify)
		aggs = append(aggs, agg)
	}
	return aggs

}

// IsIncluded returns true if the index is contained in the array
func IsIncluded(arr []int, v int) bool {
	for _, a := range arr {
		if v == a {
			return true
		}
	}
	return false
}

func toLibNodes(c lib.Constructor, nr []*lib.NodeRecord) ([]*lib.Node, error) {
	n := len(nr)
	nodes := make([]*lib.Node, n)
	var err error
	//fmt.Printf("toLibNodes: ")
	for i, record := range nr {
		nodes[i], err = record.ToNode(c)
		//fmt.Printf("\t-%d: %s \n ", i, nodes[i].Identity.PublicKey().String())
		if err != nil {
			return nil, err
		}
	}
	//fmt.Printf("\n")
	return nodes, nil
}

func requireNil(err error) {
	if err != nil {
		panic(err)
	}
}

func extractResendPeriod(opts Opts) time.Duration {
	str, ok := opts.String("ResendPeriod")
	if !ok {
		str = "200ms"
	}
	t, err := time.ParseDuration(str)
	if err != nil {
		panic(err)
	}
	return t
}

func extractAggTechnique(opts Opts) bool {
	var out bool
	v, ok := opts.Int("AggAndVerify")
	if !ok {
		v = 0
	}
	if v == 0 {
		out = false
	} else {
		out = true
	}
	return out
}
