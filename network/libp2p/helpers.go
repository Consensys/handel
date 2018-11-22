package libp2p

import (
	"fmt"
	mrand "math/rand"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

//Extract peerid and address from target
//Example
//target:  /ip4/127.0.0.1/tcp/3000/ipfs/QmQW5383sACDThGZkzCtuhbBixS8bepkPJs4dg3fAZc1Qt
//returns: <peer.ID Qm*AZc1Qt> and /ip4/127.0.0.1/tcp/3000
func makePeerIDAndAddr(target string) (*peer.ID, *multiaddr.Multiaddr, error) {
	ipfsaddr, err := multiaddr.NewMultiaddr(target)

	if err != nil {
		return nil, nil, err
	}

	pid, err := ipfsaddr.ValueForProtocol(addrType)
	if err != nil {
		return nil, nil, err
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		return nil, nil, err
	}

	targetPeerAddr, err := multiaddr.NewMultiaddr(
		fmt.Sprintf("/%s/%s", addrTypeStr, pid))

	if err != nil {
		return nil, nil, err
	}
	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	return &peerid, &targetAddr, nil
}

// MakeDeterministicID maps handel ID to libp2p ID.
// libp2p creates ID based on host private key, in general setting deterministic
// private key is not secure but in handel case we don't use any encryption, the private key
// is only used for creating libp2p identity.
// Going from handel id to libp2p id is very cumbersome, and ideally the two ids should be the same.
// Unfortunetly at this moment libp2p.Host doesn't provide option for setting up custom id.
// Note: libp2p swarm accepts custom id but for know this is overkill
func MakeDeterministicID(id int32) (crypto.PrivKey, error) {
	r := mrand.New(mrand.NewSource(int64(id)))
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	return prvKey, err
}
