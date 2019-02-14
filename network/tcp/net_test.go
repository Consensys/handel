package tcp

import (
	"testing"
	"time"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/stretchr/testify/require"
)

func TestTCPNetwork(t *testing.T) {
	addr1 := "127.0.0.1:5000"
	addr2 := "127.0.0.1:5001"
	n1, err := NewNetwork(addr1, network.NewGOBEncoding())
	require.NoError(t, err)
	n2, err := NewNetwork(addr2, network.NewGOBEncoding())
	require.NoError(t, err)

	defer n1.Stop()
	defer n2.Stop()

	received := make(chan bool, 1)
	n2.RegisterListener(handel.ListenFunc(func(p handel.ApplicationPacket) {
		received <- true
	}))

	id2 := handel.NewStaticIdentity(2, addr2, nil)
	go n1.Send([]handel.Identity{id2}, &handel.Packet{Origin: 2})

	select {
	case <-received:
	case <-time.After(500 * time.Millisecond):
		t.Fail()
	}
}
