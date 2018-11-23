package libp2p

import (
	"testing"

	h "github.com/ConsenSys/handel"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

type MockListener struct{}

func (MockListener) NewPacket(packet *h.Packet) error {
	return nil
}

func newMockListener() h.Listener {
	return MockListener{}
}

func TestRegisterListener(t *testing.T) {
	var net = NewLibP2pNetwork(0, 0)
	require.Equal(t, 0, len(*net.listeners))

	net.RegisterListener(newMockListener())
	net.RegisterListener(newMockListener())

	require.Equal(t, 2, len(*net.listeners))
}

func TestHostId(t *testing.T) {
	host1 := newHost(5000, 1)
	host2 := newHost(5000, 2)
	host3 := newHost(5000, 1)
	require.Equal(t, host1.ID().Pretty(), host3.ID().Pretty())
	require.NotEqual(t, host1.ID().Pretty(), host2.ID().Pretty())
}

func TestAddr(t *testing.T) {
	mAddr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5000")
	hAdds := hostMultiAddr("QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX", mAddr)
	result := "/ip4/127.0.0.1/tcp/5000/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX"
	require.Equal(t, hAdds.String(), result)
}
