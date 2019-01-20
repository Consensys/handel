package handel

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var msg = []byte("Sun is Shining...")


type handelTest struct {
	n        int
	offlines []int32
	thr      int
	fail     bool
}

func TestHandelTestNetworkSimple(t *testing.T) {
	var tests = []handelTest{
		{5, nil, 0, false},
	}
	testHandelTestNetwork(t, tests)
}

func TestHandelWithFailures(t *testing.T) {
	off := func(ids ...int32) []int32 {
		return ids
	}

	offs := off(0, 1, 4, 6, 10, 14, 19, 24, 27, 57, 89, 96, 101, 134, 141, 178, 179, 199, 200, 211, 243, 255, 288, 301)
	var tests = []handelTest{
		{333, offs, 333 - len(offs), false},
	}
	testHandelTestNetwork(t, tests)
}


func TestHandelTestNetworkSNonPowerOfTwo(t *testing.T) {
	off := func(ids ...int32) []int32 {
		return ids
	}

	var tests = []handelTest{
		{5, off(0), 4, false},
	}
	testHandelTestNetwork(t, tests)
}

func TestHandelTestNetworkFull(t *testing.T) {
	off := func(ids ...int32) []int32 {
		return ids
	}
	off()
	var tests = []handelTest{
		{5, off(), 5, false},
		{11, nil, 0, false},
		{33, nil, 33, false},
		{67, off(), 67, false},
		{5, off(4), 4, false},
		{13, off(0, 1, 4, 6), 6, false},
		{128, off(0, 1, 4, 6), 124, false},
		{10, off(0, 3, 5, 7, 9), 5, false},
	}
	testHandelTestNetwork(t, tests)
}

func TestHandelTestNetworkLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large test")
	}
	off := func(ids ...int32) []int32 {
		return ids
	}
	off()

	var tests = []handelTest{
		{333, off(0, 1, 4, 6, 7, 19, 56, 89, 99), 310, false},
	}
	testHandelTestNetwork(t, tests)
}

func testHandelTestNetwork(t *testing.T, tests []handelTest) {
	off := func(ids ...int32) []int32 {
		return ids
	}
	off()

	for i, scenario := range tests {
		t.Logf(" -- test %d --", i)
		n := scenario.n
		config := DefaultConfig(n)
		// When there is no offline nodes we should not rely on the timeouts
		//  for this reason we use a very long one, so the tests will fail with a timeout
		//  if there is a bug.
		if len(scenario.offlines) == 0 {
			config.NewTimeoutStrategy = NewInfiniteTimeout
		}
		secrets := make([]SecretKey, n)
		pubs := make([]PublicKey, n)
		cons := new(fakeCons)
		for i := 0; i < n; i++ {
			secrets[i] = new(fakeSecret)
			pubs[i] = &fakePublic{true}
		}
		test := NewTest(secrets, pubs, cons, msg, config)
		if scenario.thr != 0 {
			test.SetOfflineNodes(scenario.offlines...)
			test.SetThreshold(scenario.thr)
		}
		test.Start()
		select {
		case <-test.WaitCompleteSuccess():
			// all good
			fmt.Printf("*** sent=%d, rcv=%d\n", test.handels[0].stats.msgSentCt, test.handels[0].stats.msgRcvCt)
		case <-time.After(100 * time.Second):
			if scenario.fail {
				continue
			}
			t.FailNow()
		}
		test.Stop()
	}
}

func TestHandelWholeThing(t *testing.T) {
	//t.Skip()
	n := 32
	reg, handels := FakeSetup(n)
	defer CloseHandels(handels)
	//PrintLog = false
	t.Logf("%d", reg.Size())
	for _, h := range handels {
		go h.Start()
	}

	type sigTest struct {
		sender *Handel
		ms     *MultiSignature
	}

	var wg sync.WaitGroup
	wg.Add(n)
	verif := make(chan sigTest, n)
	doneCh := make([]chan bool, n)
	for i := 0; i < n; i++ {
		doneCh[i] = make(chan bool, 10)
	}

	//var cc int32
	for _, h := range handels {
		go func(hh *Handel) {
			var wgDone bool
			id := hh.id.ID()
			for {
				select {
				case ms := <-hh.FinalSignatures():
					if !wgDone {
						//c := atomic.AddInt32(&cc, 1)
						//fmt.Printf(" +++ TEST - HANDEL %d FINISHED %d/%d+++ sig %d\n", id, c, n, ms.Cardinality())
						wg.Done()
						wgDone = true
					} else {
						//fmt.Printf(" +++ TEST - HANDEL %d FINISHED -> sig %d +++ \n", id, ms.Cardinality())
					}
					verif <- sigTest{ms: &ms, sender: hh}
				case <-doneCh[id]:
					return
				}
			}
		}(h)
	}

	wg.Wait()

	var counter int
	var handelsDone = make([]bool, n)

	checkAllDone := func() bool {
		for _, d := range handelsDone {
			if !d {
				return false
			}
		}
		return true
	}

	for st := range verif {
		counter++
		id := st.sender.id.ID()
		if handelsDone[id] {
			continue
		}

		if st.ms.Cardinality() == n {
			handelsDone[st.sender.id.ID()] = true
			doneCh[id] <- true
		}

		if checkAllDone() {
			break
		}
	}

	require.True(t, counter >= n)
}

func TestHandelCheckCompletedLevel(t *testing.T) {
	n := 8
	_, handels := FakeSetup(n)
	defer CloseHandels(handels)

	// simulate not-complete signature of level 1 on node 1
	// checkCompletedLevel should not react in any way
	sender := handels[1]
	receiver2 := handels[2]
	inc2 := make(chan *Packet)
	receiver2.net.(*TestNetwork).lis = []Listener{ChanListener(inc2)}

	sig0 := fullIncomingSig(1)
	// not-complete signature
	sig02 := fullIncomingSig(2)
	sig02.ms.BitSet.Set(0, true)

	// node 0 corresponds to level 1 in node's 1 view.
	// => store incomplete signature as if it was an empty signature from node 0
	// node 1 should NOT send anything to node 2 (or 3 but we're only verifying
	// node 2 since it will send to both anyway)
	sender.store.Store(sig02)
	sender.checkCompletedLevel(sig02)
	select {
	case <-inc2:
		t.Fatal("should not have received anything")
	case <-time.After(20 * time.Millisecond):
		// good
	}

	// send full signature
	// node 2 should react
	sender.store.Store(sig0)
	sender.checkCompletedLevel(sig0)
	select {
	case p := <-inc2:
		require.Equal(t, int32(1), p.Origin)
		require.Equal(t, byte(2), p.Level)
	case <-time.After(20 * time.Millisecond):
		t.Fatal("not received expected full signature")
	}
}

func TestHandelCheckFinalSignature(t *testing.T) {
	n := 16

	type checkFinalTest struct {
		// one slice represents sigs to store before calling the checkVerified
		// you can put multiple slices to call checkverified multiple times
		sigs [][]*incomingSig
		// input to the handler
		input *incomingSig
		// expected output on the output channel
		out []*MultiSignature
	}

	// test(3) set a non-complete signature followed by a complete signature
	pairs1 := incomingSigs(0, 1, 2, 3, 4)
	pairs2 := incomingSigs(3, 4)
	// index 8 (2^4-1) + 6 = 14 set to false
	pairs1[4].ms.BitSet.Set(6, false)
	final4 := finalIncomingSig(4, n)
	// missing one contribution
	final4b := finalIncomingSig(4, n)
	final4b.ms.BitSet.Set(14, false)

	// test(4) set a under-threshold signature followed by a good one
	pairs3 := incomingSigs(0, 1, 2, 3, 4)
	// 4-1 -> because that's how you compute the size of a level
	// -1 -> to just spread out holes to other levels and leave this one still
	// having one contribution
	for i := 0; i < pow2(4-1)-1; i++ {
		pairs3[4].ms.BitSet.Set(i, false)
	}
	pairs3[3].ms.BitSet.Set(1, false)
	pairs3[3].ms.BitSet.Set(2, false)

	//fmt.Println("pairs3[4] bitset = ", pairs3[4].ms.BitSet.String())
	//fmt.Println("pairs3[3] bitset = ", pairs3[3].ms.BitSet.String())

	toMatrix := func(pairs ...[]*incomingSig) [][]*incomingSig {
		return append(make([][]*incomingSig, 0), pairs...)
	}
	var tests = []checkFinalTest{
		// too lower level signatures
		{toMatrix(incomingSigs(0, 1, 2)), nil, []*MultiSignature{nil}},
		// everything's perfect
		{toMatrix(incomingSigs(0, 1, 2, 3, 4)), nil, []*MultiSignature{finalIncomingSig(4, n).ms}},
		// gives two consecutives better
		{toMatrix(pairs1, pairs2), nil, []*MultiSignature{final4b.ms, final4.ms}},
		// one underthreshold and fully signed
		{toMatrix(pairs3, pairs2), nil, []*MultiSignature{nil, final4.ms}},
	}

	waitOut := func(h *Handel) *MultiSignature {
		select {
		case ms := <-h.FinalSignatures():
			return &ms
		case <-time.After(20 * time.Millisecond):
			return nil
		}
	}

	for i, test := range tests {
		t.Logf(" -- test %d --", i)
		_, handels := FakeSetup(n)
		h := handels[1]
		store := h.store

		for i, toInsert := range test.sigs {
			// insert slice
			for _, sig := range toInsert {
				store.Store(sig)
			}
			h.checkFinalSignature(test.input)

			// lookup expected result at that point
			expected := test.out[i]
			output := waitOut(h)
			require.Equal(t, expected, output)
		}
	}
}

func TestHandelParsePacket(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	//ids := registry.(*arrayRegistry).ids // TODO: The test runs ok even if we comment this lines
	c := DefaultConfig(n)
	c.DisableShuffling = true
	h := &Handel{
		c:           c,
		reg:         registry,
		cons:        new(fakeCons),
		msg:         msg,
		Partitioner: NewBinPartitioner(1, registry),
	}
	h.levels = createLevels(h.c, h.Partitioner)
	type packetTest struct {
		*Packet
		Error bool
	}
	correctSig := newSig(fullBitset(2))
	buffMs, _ := correctSig.MarshalBinary()
	incorrectSig := newSig(fullBitset(5))
	invalidMsBuff, _ := incorrectSig.MarshalBinary()
	packets := []*packetTest{
		{
			&Packet{
				Origin:   65000,
				Level:    0,
				MultiSig: fakeConstSig,
			}, true,
		},
		{
			&Packet{
				Origin:   3,
				Level:    254,
				MultiSig: fakeConstSig,
			}, true,
		},
		{
			&Packet{
				Origin:   3,
				Level:    1,
				MultiSig: []byte{0x01},
			}, true,
		},
		{
			&Packet{
				Origin:   3,
				Level:    2,
				MultiSig: buffMs,
			}, false,
		},
		{
			&Packet{
				Origin:   3,
				Level:    2,
				MultiSig: invalidMsBuff,
			}, true,
		},
	}
	for i, test := range packets {
		t.Logf(" -- test %d --", i)
		err := h.validatePacket(test.Packet)
		_, _, err2 := h.parseSignatures(test.Packet)
		if test.Error {
			var isErr1 = err != nil
			var isErr2 = err2 != nil
			require.True(t, isErr1 || isErr2)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestHandelCreateLevel(t *testing.T) {
	n := 16
	registry := FakeRegistry(n)
	part := NewBinPartitioner(1, registry)
	c := DefaultConfig(n)
	c.DisableShuffling = true

	mapping1 := createLevels(c, part)
	mapping2 := createLevels(c, part)
	require.Equal(t, mapping1, mapping2)

	seed := make([]byte, 512)
	_, err := rand.Reader.Read(seed)
	require.NoError(t, err)

	c.DisableShuffling = false
	var r bytes.Buffer
	r.Write(seed)
	c.Rand = &r
	mapping3 := createLevels(c, part)
	require.NotEqual(t, mapping3, mapping2)

	var r2 bytes.Buffer
	r2.Write(seed)
	c.Rand = &r2
	mapping4 := createLevels(c, part)
	require.Equal(t, mapping3, mapping4)

	c = DefaultConfig(n)
	mapping5 := createLevels(c, part)
	require.NotEqual(t, mapping5, mapping4)
	require.NotEqual(t, mapping5, mapping1)
}
