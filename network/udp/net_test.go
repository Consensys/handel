package udp

import (
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/stretchr/testify/require"
)

func TestUDPNetwork(t *testing.T) {
	n1, err := NewNetwork("127.0.0.1:3000", network.NewGOBEncoding())
	require.NoError(t, err)
	n2, err := NewNetwork("127.0.0.1:3001", network.NewGOBEncoding())
	require.NoError(t, err)

	received := make(chan bool, 1)
	n2.RegisterListener(handel.ListenFunc(func(p *handel.Packet) {
		received <- true
	}))

	id2 := handel.NewStaticIdentity(2, "127.0.0.1:3001", nil)
	n1.Send([]handel.Identity{id2}, &handel.Packet{Origin: 2, MultiSig: []byte{0x01}})

	select {
	case <-received:
	case <-time.After(500 * time.Millisecond):
		t.Fail()
	}
}
