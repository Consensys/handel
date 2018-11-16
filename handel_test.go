package handel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var msg = []byte("Sun is Shining...")

func TestHandelcheckFinalSignature(t *testing.T) {
	n := 16

	type checkFinalTest struct {
		// one slice represents sigs to store before calling the checkVerified
		// you can put multiple slices to call checkverified multiple times
		sigs [][]*sigPair
		// input to the handler
		input *verifiedSig
		// expected output on the output channel
		out []*MultiSignature
	}

	pairs1 := sigPairs(0, 1, 2, 3, 4)
	pairs2 := sigPairs(4)

	// set a non-complete signature
	// index 8 (2^4-1) + 6 = 14 set to false
	pairs1[4].ms.BitSet.Set(6, false)

	final4 := finalSigPair(4, n)
	// missing one contribution
	final4b := finalSigPair(4, n)
	final4b.ms.BitSet.Set(14, false)

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

func TestHandelVerifySignature(t *testing.T) {
	/*n := 16*/

	//type sigTest struct {
	//changeIDs func(ids []Identity)
	//ms        *MultiSignature
	//origin    uint32
	//level     byte
	//isErr     bool
	//}

	//// helper functions to manipulate identities
	//allNotVerify := func(ids []Identity) {
	//for _, i := range ids {
	//i.(*fakeIdentity).fakePublic.verify = false
	//}
	//}
	//idempotent := func(ids []Identity) {}
	//var sigTests = []sigTest{
	//// everything's good
	//{idempotent, newSig(fullBitset(2)), 3, 2, false},
	//// just invalid sig
	//{allNotVerify, newSig(fullBitset(2)), 3, 2, true},
	//// invalid level value
	//{allNotVerify, newSig(fullBitset(2)), 3, 0, true},
	//// wrong origin value -- too high
	//{allNotVerify, newSig(fullBitset(2)), 7, 2, true},
	//// wrong origin value -- too low
	//{allNotVerify, newSig(fullBitset(2)), 0, 2, true},
	//// wrong bitset length
	//{allNotVerify, newSig(fullBitset(3)), 3, 2, true},
	//// invalid individual signature
	//{func(ids []Identity) {
	//// invalid signature from node in the expected bitset
	//ids[3].(*fakeIdentity).fakePublic.verify = false
	//}, newSig(fullBitset(2)), 3, 2, true},
	//}

	//for _, test := range sigTests {
	//registry := FakeRegistry(n)
	//ids := registry.(*arrayRegistry).ids
	//h := &Handel{
	//c:      DefaultConfig(n),
	//reg:    registry,
	//cons: new(fakeScheme),
	//msg:    msg,
	//tree:   newCandidateTree(ids[1].ID(), registry),
	//}
	//test.changeIDs(ids)
	//err := h.verifySignature(test.ms, test.origin, test.level)
	//if test.isErr {
	//require.Error(t, err)
	//continue
	//}
	//require.NoError(t, err)
	/*}*/
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
