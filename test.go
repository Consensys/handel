package handel

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Test is a struct implementing some useful functionality to test specific
// implementations on Handel
type Test struct {
	reg     Registry
	nets    []Network
	handels []*Handel
	// notifies when one handel instance have finished
	finished chan int
	// mapping of the finished handel instances
	completed map[int]bool
	// notifies when the test should be brought down
	done chan bool
	// complete success channel gets notified when all handel instances have
	// output a complete multi-signature
	completeSuccess chan bool
	// list of IDs that are offline during the test
	offline []int32
	// threshold of contributions necessary
	threshold int
}

// NewTest returns all handels instances ready to go !
func NewTest(keys []SecretKey, pubs []PublicKey, c Constructor, msg []byte) *Test {
	n := len(keys)
	ids := make([]Identity, n)
	sigs := make([]Signature, n)
	nets := make([]Network, n)
	handels := make([]*Handel, n)
	var err error
	for i := 0; i < n; i++ {
		pk := pubs[i]
		id := int32(i)
		ids[i] = NewStaticIdentity(id, "", pk)
		sigs[i], err = keys[i].Sign(msg, rand.Reader)
		if err != nil {
			panic(err)
		}
		nets[i] = &TestNetwork{id: id, list: nets}
	}
	reg := NewArrayRegistry(ids)
	for i := 0; i < n; i++ {
		handels[i] = NewHandel(nets[i], reg, ids[i], c, msg, sigs[i])
	}
	return &Test{
		reg:             reg,
		nets:            nets,
		handels:         handels,
		done:            make(chan bool),
		finished:        make(chan int, n),
		completed:       make(map[int]bool),
		completeSuccess: make(chan bool, 1),
		offline:         make([]int32, 0),
		threshold:       n,
	}
}

// SetOfflineNodes sets the given list of node's ID as offline nodes - the
// designated nodes won't run during the simulation.
func (t *Test) SetOfflineNodes(ids ...int32) {
	t.offline = append(t.offline, ids...)
}

// SetThreshold sets the minimum threshold of contributions required to be
// present in the multisignature created by Handel nodes. By default, it is
// equal to the size of the participant's set.
func (t *Test) SetThreshold(threshold int) {
	t.threshold = threshold
}

// Start manually every handel instances and starts go routine to listen to the
// final signatures output from the handel instances.
func (t *Test) Start() {
	for i, handel := range t.handels {
		if t.isOffline(handel.id.ID()) {
			continue
		}
		idx := i
		go handel.Start()
		go t.waitFinalSig(idx)
	}
	go t.watchComplete()
}

func (t *Test) isOffline(nodeID int32) bool {
	for _, id := range t.offline {
		if id == nodeID {
			return true
		}
	}
	return false
}

// Stop manually every handel instances
func (t *Test) Stop() {
	close(t.done)
	time.Sleep(30 * time.Millisecond)
	for _, handel := range t.handels {
		handel.Stop()
	}
}

// Networks returns the slice of network interface used by handel. It can be
// useful if you want to register your own listener.
func (t *Test) Networks() []Network {
	return t.nets
}

// WaitCompleteSuccess waits until *all* handel instance have generated the
// multi-signature containing *all* contributions from each. It returns an
// channel so it's easy to wait for a certain timeout with `select`.
func (t *Test) WaitCompleteSuccess() chan bool {
	return t.completeSuccess
}

func (t *Test) watchComplete() {
	for {
		select {
		case i := <-t.finished:
			t.completed[i] = true
			fmt.Printf("\n +++ %s +++\n\n", t.String())
			if t.allCompleted() {
				// signature that to success channel
				t.completeSuccess <- true
				return
			}
		case <-t.done:
			return
		}
	}
}

// waitFinalSig loops over the final signatures output by a specific handel
// instance until the signature is complete. In that case, it notifies the main
// watch routine.
func (t *Test) waitFinalSig(i int) {
	h := t.handels[i]
	ch := h.FinalSignatures()
	for {
		select {
		case ms := <-ch:
			/*fmt.Println("+++++++ t.reg ", t.reg)*/
			//fmt.Println("+++++++ ms", ms)
			/*fmt.Println("+++++++ ms.BitSet ", ms.BitSet)*/
			if ms.BitSet.Cardinality() >= t.threshold {
				// one full !
				t.finished <- i
				return
			}
		case <-t.done:
			return
		}
	}
}

func (t *Test) allCompleted() bool {
	for _, f := range t.completed {
		if !f {
			return false
		}
	}
	return true
}

func (t *Test) String() string {
	count := 0
	for _, f := range t.completed {
		if f {
			count++
		}
	}
	online := len(t.handels) - len(t.offline)
	return fmt.Sprintf("test network - finished %d / online %d / total %d", count, online, len(t.handels))
}

// TestNetwork is a simple Network implementation using local dispatch functions
// in goroutine.
type TestNetwork struct {
	id   int32
	list []Network
	lis  []Listener
}

// Send implements the Network interface
func (f *TestNetwork) Send(ids []Identity, p *Packet) {
	for _, id := range ids {
		go func(i Identity) {
			f.list[int(i.ID())].(*TestNetwork).dispatch(p)
		}(id)
	}
}

// RegisterListener implements the Network interface
func (f *TestNetwork) RegisterListener(l Listener) {
	f.lis = append(f.lis, l)
}

func (f *TestNetwork) dispatch(p *Packet) {
	for _, l := range f.lis {
		l.NewPacket(p)
	}
}
