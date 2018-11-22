package handel

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var msg = []byte("Sun is Shining...")

func TestHandelWholeThing(t *testing.T) {
	//t.Skip()
	n := 16
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

	for _, h := range handels {
		go func(hh *Handel) {
			var wgDone bool
			id := hh.id.ID()
			for {
				select {
				case ms := <-hh.FinalSignatures():
					if !wgDone {
						wg.Done()
						wgDone = true
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

func TestHandelcheckCompletedLevel(t *testing.T) {
	n := 8
	_, handels := FakeSetup(n)
	defer CloseHandels(handels)

	// 1 should send to 2 only a full signature
	sender := handels[1]
	receiver2 := handels[2]
	receiver4 := handels[4]
	inc2 := make(chan *Packet)
	receiver2.net.(*fakeNetwork).lis = []Listener{ChanListener(inc2)}
	inc4 := make(chan *Packet)
	receiver4.net.(*fakeNetwork).lis = []Listener{ChanListener(inc4)}

	sig2 := fullSigPair(2)
	// not-complete signature
	sig22 := fullSigPair(2)
	sig22.ms.BitSet.Set(0, false)

	// send not full signature
	sender.store.Store(2, sig22.ms)
	sender.checkCompletedLevel(sig22)
	select {
	case <-inc2:
		t.Fatal("should not have received anything")
	case <-time.After(20 * time.Millisecond):
		// good
	}

	// send full signature
	sender.store.Store(2, sig2.ms)
	sender.store.Store(1, fullSigPair(1).ms)
	sender.store.Store(0, fullSigPair(1).ms)
	sender.checkCompletedLevel(sig2)
	select {
	case p := <-inc4:
		require.Equal(t, int32(1), p.Origin)
		require.Equal(t, byte(2), p.Level)
	case <-time.After(20 * time.Millisecond):
		t.Fatal("not received expected full signature")
	}
}

func TestHandelcheckFinalSignature(t *testing.T) {
	n := 16

	type checkFinalTest struct {
		// one slice represents sigs to store before calling the checkVerified
		// you can put multiple slices to call checkverified multiple times
		sigs [][]*sigPair
		// input to the handler
		input *sigPair
		// expected output on the output channel
		out []*MultiSignature
	}

	// test(3) set a non-complete signature followed by a complete signature
	pairs1 := sigPairs(0, 1, 2, 3, 4)
	pairs2 := sigPairs(3, 4)
	// index 8 (2^4-1) + 6 = 14 set to false
	pairs1[4].ms.BitSet.Set(6, false)
	final4 := finalSigPair(4, n)
	// missing one contribution
	final4b := finalSigPair(4, n)
	final4b.ms.BitSet.Set(14, false)

	// test(4) set a under-threshold signature followed by a good one
	pairs3 := sigPairs(0, 1, 2, 3, 4)
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

	toMatrix := func(pairs ...[]*sigPair) [][]*sigPair {
		return append(make([][]*sigPair, 0), pairs...)
	}
	var tests = []checkFinalTest{
		// too lower level signatures
		{toMatrix(sigPairs(0, 1, 2)), nil, []*MultiSignature{nil}},
		// everything's perfect
		{toMatrix(sigPairs(0, 1, 2, 3, 4)), nil, []*MultiSignature{finalSigPair(4, n).ms}},
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
				store.Store(sig.level, sig.ms)
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
	ids := registry.(*arrayRegistry).ids
	h := &Handel{
		c:    DefaultConfig(n),
		reg:  registry,
		cons: new(fakeCons),
		msg:  msg,
		part: newBinTreePartition(ids[1].ID(), registry),
	}
	type packetTest struct {
		*Packet
		Error bool
	}
	correctSig := newSig(fullBitset(2))
	buffMs, _ := correctSig.MarshalBinary()
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
	}
	for _, test := range packets {
		_, err := h.parsePacket(test.Packet)
		if test.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
