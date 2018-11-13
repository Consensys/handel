package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandelParsePacket(t *testing.T) {
	n := 17
	registry := FakeRegistry(n)

	msg := []byte("Sun is Shining...")

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
