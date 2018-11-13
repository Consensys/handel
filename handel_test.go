package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var msg = []byte("Sun is Shining...")

func TestHandelVerifySignature(t *testing.T) {
	n := 16

	type sigTest struct {
		changeIDs func(ids []Identity)
		ms        *MultiSignature
		origin    uint32
		level     byte
		isErr     bool
	}

	// helper functions to manipulate identities
	idempotent := func(ids []Identity) {}
	allVerify := func(ids []Identity) {
		for _, i := range ids {
			i.(*fakeIdentity).fakePublic.verify = true
		}
	}
	// returns a multisignature from a bitset
	newSig := func(b BitSet) *MultiSignature {
		return &MultiSignature{
			BitSet:    b,
			Signature: new(fakeSig),
		}
	}
	var sigTests = []sigTest{
		// everything's good
		{allVerify, newSig(fullBitset(2)), 3, 2, false},
		// just invalid sig
		{idempotent, newSig(fullBitset(2)), 3, 2, true},
		// invalid level value
		{allVerify, newSig(fullBitset(2)), 3, 0, true},
		// wrong origin value -- too high
		{allVerify, newSig(fullBitset(2)), 7, 2, true},
		// wrong origin value -- too low
		{allVerify, newSig(fullBitset(2)), 0, 2, true},
		// wrong bitset length
		{allVerify, newSig(fullBitset(3)), 3, 2, true},
		// invalid individual signature
		{func(ids []Identity) {
			allVerify(ids)
			// invalid signature from node in the expected bitset
			ids[3].(*fakeIdentity).fakePublic.verify = false
		}, newSig(fullBitset(2)), 3, 2, true},
	}

	for _, test := range sigTests {
		registry := FakeRegistry(n)
		ids := registry.(*arrayRegistry).ids
		h := &Handel{
			c:      DefaultConfig(n),
			reg:    registry,
			scheme: new(fakeScheme),
			msg:    msg,
			tree:   newCandidateTree(ids[1].ID(), registry),
		}
		test.changeIDs(ids)
		err := h.verifySignature(test.ms, test.origin, test.level)
		if test.isErr {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
	}
}

func TestHandelParsePacket(t *testing.T) {
	n := 17
	registry := FakeRegistry(n)

	h := &Handel{
		c:      DefaultConfig(n),
		reg:    registry,
		scheme: new(fakeScheme),
		msg:    msg,
	}

	type packetTest struct {
		*Packet
		Error bool
	}
	correctMs := &MultiSignature{
		BitSet:    NewWilffBitset(10),
		Signature: new(fakeSig),
	}
	buffMs, _ := correctMs.MarshalBinary()
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
				Level:    1,
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
