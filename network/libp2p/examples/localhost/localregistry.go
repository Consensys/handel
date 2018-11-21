package localhost

import (
	"fmt"

	h "github.com/ConsenSys/handel"
	peer "github.com/libp2p/go-libp2p-peer"

	l "github.com/ConsenSys/handel/network/libp2p"
)

type staticRegistry struct {
	size int
	ids  []h.Identity
}

type LibP2PIdentity struct {
	address string
	id      int32
}

func NewLibP2PIdentity(address string, id int32) LibP2PIdentity {
	return LibP2PIdentity{address: address, id: id}
}

func (identity LibP2PIdentity) Address() string {
	return identity.address
}

func (identity LibP2PIdentity) PublicKey() h.PublicKey {
	return nil
}

func (identity LibP2PIdentity) ID() int32 {
	return identity.id
}

func NewLocalStaticRegistry(size int32) h.Registry {
	ids := []h.Identity{}
	for idx := int32(0); idx < size; idx++ {
		privKey, _ := l.MakeDeterministicID(idx)

		pid, _ := peer.IDFromPrivateKey(privKey)
		peerid := peer.IDB58Encode(pid)

		addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/ipfs/%s", 3000+idx, peerid)

		id := LibP2PIdentity{address: addr, id: idx}
		ids = append(ids, id)
	}
	return h.NewArrayRegistry(ids)
}
