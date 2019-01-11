package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/bn256"
	"github.com/ConsenSys/handel/simul/lib"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	p2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

const topicName = "handel"
const ping = "/echo/1.0.0"

// P2PIdentity represents the public side of a node within the libp2p gossip
// context
type P2PIdentity struct {
	h.Identity
	id   peer.ID
	addr ma.Multiaddr
}

// NewP2PIdentity returns the public side of gossip node - useful for connecting
// to them
func NewP2PIdentity(id h.Identity) (*P2PIdentity, error) {
	pub := &bn256Pub{id.PublicKey().(*bn256.PublicKey)}
	fullAddr := id.Address()
	ip, port, err := net.SplitHostPort(fullAddr)
	if err != nil {
		return nil, err
	}
	multiAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", ip, port))
	if err != nil {
		return nil, err
	}
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return &P2PIdentity{
		Identity: id,
		id:       pid,
		addr:     multiAddr,
	}, nil
}

// P2PNode represents the libp2p version of a node - with a host.Host and
// pubsub.PubSub structure.
type P2PNode struct {
	handelID int32
	priv     *bn256Priv
	h        host.Host
	g        *pubsub.PubSub
	s        *pubsub.Subscription
}

// NewP2PNode transforms a lib.Node to a p2p node.
func NewP2PNode(handelNode *lib.Node) (*P2PNode, error) {
	secret := handelNode.SecretKey.(*bn256.SecretKey)
	pub := handelNode.Identity.PublicKey().(*bn256.PublicKey)
	priv := &bn256Priv{
		SecretKey: secret,
		pub:       &bn256Pub{pub},
	}
	fullAddr := handelNode.Address()
	ip, port, err := net.SplitHostPort(fullAddr)
	if err != nil {
		return nil, err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%s", ip, port)),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
		libp2p.NoSecurity,
	}
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	basicHost.SetStreamHandler(ping, func(s p2pnet.Stream) {
		if err := pong(s); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	// needed to run in insecure mode... ><
	basicHost.Peerstore().AddPubKey(basicHost.ID(), priv.GetPublic())

	//manager := basicHost.ConnManager()
	////bundle := manager.Notifee().(*p2pnet.NotifyBundle)
	//fmt.Println(bundle)

	// create the pubsub struct
	opt := pubsub.WithMessageSigning(false)
	gossip, err := pubsub.NewGossipSub(context.Background(), basicHost, opt)
	//gossip, err := pubsub.NewFloodSub(context.Background(), basicHost, opt)
	if err != nil {
		return nil, err
	}

	subscription, err := gossip.Subscribe(topicName)
	return &P2PNode{
		handelID: handelNode.Identity.ID(),
		priv:     priv,
		h:        basicHost,
		g:        gossip,
		s:        subscription,
	}, err
}

// Connect to the given identity
func (p *P2PNode) Connect(p2 *P2PIdentity) error {
	p.h.Peerstore().AddAddr(p2.id, p2.addr, pstore.PermanentAddrTTL)
	return p.ping(p2)
	//return p.h.Connect(context.Background(), p.h.Peerstore().PeerInfo(p2.id))
}

// Gossip broadcasts the given message to the overlay network
func (p *P2PNode) Gossip(msg []byte) error {
	return p.g.Publish(topicName, msg)
}

// Next returns the next item under the registered topic
func (p *P2PNode) Next() ([]byte, error) {
	pbMsg, err := p.s.Next(context.Background())
	if err != nil {
		return nil, err
	}
	//fmt.Printf("p2pnode %d - new message from %s\n", p.handelID, string(pbMsg.From))
	return pbMsg.Data, nil
}

func (p *P2PNode) ping(p2 *P2PIdentity) error {
	s, err := p.h.NewStream(context.Background(), p2.id, ping)
	if err != nil {
		log.Fatalln(err)
	}

	msg := []byte("HelloWorld\n")
	_, err = s.Write(msg)
	if err != nil {
		log.Fatalln(err)
	}

	out, err := ioutil.ReadAll(s)
	if err != nil {
		log.Fatalln(err)
	}
	if string(out) != string(msg) {
		return errors.New("ping/pong failed")
	}
	return nil
}

func pong(s p2pnet.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	_, err = s.Write([]byte(str))
	return err
}

// P2PRegistry is a handel.Registry with a list of P2pIdentity as a backend
type P2PRegistry []*P2PIdentity

// Size implements the handel.Registry interface
func (p *P2PRegistry) Size() int {
	return len(*p)
}

// Identity implements the handel.Registry interface
func (p *P2PRegistry) Identity(idx int) (handel.Identity, bool) {
	if idx < 0 || idx >= p.Size() {
		return nil, false
	}
	return (*p)[idx], true
}

// Identities implements the handel.Registry interface
func (p *P2PRegistry) Identities(from, to int) ([]handel.Identity, bool) {
	if !p.inBound(from) || !p.inBound(to) {
		return nil, false
	}
	if to < from {
		return nil, false
	}
	arr := (*p)[from:to]
	ret := make([]handel.Identity, len(arr))
	for i, p := range arr {
		ret[i] = p
	}
	return ret, true
}

func (p *P2PRegistry) inBound(idx int) bool {
	return !(idx < 0 || idx > p.Size())
}
