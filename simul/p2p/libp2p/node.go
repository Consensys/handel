package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	"github.com/ConsenSys/handel/simul/lib"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	metrics "github.com/libp2p/go-libp2p-metrics"
	p2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

const topicName = "handel"
const ping = "/echo/1.0.0"

// NewRouter is a type to designate router creation factory
type NewRouter func(context.Context, host.Host, ...pubsub.Option) (*pubsub.PubSub, error)

// P2PIdentity represents the public side of a node within the libp2p gossip
// context
type P2PIdentity struct {
	h.Identity
	id   peer.ID
	addr ma.Multiaddr
}

// NewP2PIdentity returns the public side of gossip node - useful for connecting
// to them
func NewP2PIdentity(id h.Identity, c lib.Constructor) (*P2PIdentity, error) {
	pub := &bn256Pub{PublicKey: id.PublicKey().(lib.PublicKey), newSig: c.Signature}
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
	ctx          context.Context
	id           handel.Identity
	handelID     int32
	enc          network.Encoding
	priv         *bn256Priv
	h            host.Host
	g            *pubsub.PubSub
	s            *pubsub.Subscription
	reg          P2PRegistry
	ch           chan handel.Packet
	resendPeriod time.Duration
	ticker       *time.Ticker
	setup        handel.BitSet
	setupDoneCh  chan bool
	setupDone    bool
	threshold    int
	reporter     *proxyReporter
}

// NewP2PNode transforms a lib.Node to a p2p node.
func NewP2PNode(ctx context.Context, handelNode *lib.Node, n NewRouter, reg P2PRegistry, cons lib.Constructor, threshold int) (*P2PNode, error) {
	secret := handelNode.SecretKey.(lib.SecretKey)
	pub := handelNode.Identity.PublicKey().(lib.PublicKey)
	priv := &bn256Priv{
		SecretKey: secret,
		pub:       &bn256Pub{PublicKey: pub, newSig: cons.Signature},
	}
	fullAddr := handelNode.Address()
	_, port, err := net.SplitHostPort(fullAddr)
	if err != nil {
		return nil, err
	}
	reporter := newProxyReporter()
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%s", "0.0.0.0", port)),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
		libp2p.NoSecurity,
		libp2p.BandwidthReporter(reporter),
	}
	basicHost, err := libp2p.New(ctx, opts...)
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
	router, err := n(ctx, basicHost, opt)
	//gossip, err := pubsub.NewGossipSub(ctx, basicHost, opt)
	//gossip, err := pubsub.NewFloodSub(context.Background(), basicHost, opt)
	if err != nil {
		return nil, err
	}

	subscription, err := router.Subscribe(topicName)
	node := &P2PNode{
		ctx:          ctx,
		enc:          network.NewGOBEncoding(),
		handelID:     handelNode.Identity.ID(),
		id:           handelNode.Identity,
		priv:         priv,
		h:            basicHost,
		g:            router,
		s:            subscription,
		reg:          reg,
		ch:           make(chan handel.Packet, reg.Size()),
		setup:        handel.NewWilffBitset(reg.Size()),
		resendPeriod: 500 * time.Millisecond,
		setupDoneCh:  make(chan bool, 1),
		threshold:    threshold,
		reporter:     reporter,
	}
	node.setup.Set(int(node.id.ID()), true)
	go node.readNexts()
	return node, err
}

var setupPacket = &handel.Packet{Level: 255}

// WaitAllSetup periodically do the following:
// - sends out its setup message on the topic
// Until the node received all setup message from everybody expected
// The messages are read as Packets with special origin -1.
func (p *P2PNode) WaitAllSetup() chan bool {
	p.ticker = time.NewTicker(p.resendPeriod)
	ch := make(chan bool, 1)
	go func() {
		ok := true
		for ok {
			select {
			case <-p.ticker.C:
				setupPacket.Origin = p.id.ID()
				p.Diffuse(setupPacket)
			case <-p.setupDoneCh:
				ok = false
			case <-p.ctx.Done():
				ok = false
			}
		}
		ch <- true
	}()
	return ch
}

// Connect to the given identity
func (p *P2PNode) Connect(i handel.Identity) error {
	p2 := p.reg[i.ID()]
	p.h.Peerstore().AddAddr(p2.id, p2.addr, pstore.PermanentAddrTTL)
	err := p.h.Connect(context.Background(), p.h.Peerstore().PeerInfo(p2.id))
	if err != nil {
		return err
	}
	return p.ping(p2)
}

// Diffuse broadcasts the given message to the overlay network
func (p *P2PNode) Diffuse(packet *handel.Packet) {
	var b bytes.Buffer
	if err := p.enc.Encode(packet, &b); err != nil {
		fmt.Println(err)
		panic(err)
	}
	if err := p.g.Publish(topicName, b.Bytes()); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

// Identity implements the Node interface
func (p *P2PNode) Identity() handel.Identity {
	return p.id
}

// Next implements the Node interface
func (p *P2PNode) Next() chan handel.Packet {
	return p.ch
}

// SecretKey implements the Node interface
func (p *P2PNode) SecretKey() lib.SecretKey {
	return p.priv.SecretKey
}

func (p *P2PNode) readNexts() {
	for {
		pbMsg, err := p.s.Next(context.Background())
		if err != nil {
			fmt.Println(" -- err:", err)
			return
		}
		packet, err := p.enc.Decode(bytes.NewBuffer(pbMsg.Data))
		if err != nil {
			fmt.Println(" ++ err:", err)
			return
		}
		if packet.Level == setupPacket.Level {
			p.incomingSetup(packet)
		} else {
			p.ch <- *packet
		}
	}
}

func (p *P2PNode) incomingSetup(packet *handel.Packet) {
	p.setup.Set(int(packet.Origin), true)
	if p.setup.Cardinality() >= p.threshold && !p.setupDone {
		p.setupDone = true
		p.setupDoneCh <- true
	}
}

// Values implements the monitor.Counter interface
// TODO
func (p *P2PNode) Values() map[string]float64 {
	return p.reporter.Values()
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

func getRouter(opts map[string]string) NewRouter {
	str, exists := opts["Router"]
	if !exists {
		str = "flood"
	}

	var n NewRouter
	switch strings.ToLower(str) {
	case "flood":
		fmt.Println("using flood router")
		n = pubsub.NewFloodSub
	case "gossip":
		n = pubsub.NewGossipSub
	default:
		n = pubsub.NewGossipSub
	}
	return n
}

type proxyReporter struct {
	sync.Mutex
	metrics.Reporter
	bytesSent int64
	bytesRcvd int64
}

func newProxyReporter() *proxyReporter {
	return &proxyReporter{Reporter: metrics.NewBandwidthCounter()}
}

func (p *proxyReporter) LogRecvMessage(i int64) {
	p.Lock()
	p.bytesRcvd += i
	p.Unlock()
	p.Reporter.LogRecvMessage(i)
}

func (p *proxyReporter) LogSentMessage(i int64) {
	p.Lock()
	p.bytesSent += i
	p.Unlock()
	p.Reporter.LogRecvMessage(i)
}

// Values implement the monitor.Counter interface
func (p *proxyReporter) Values() map[string]float64 {
	p.Lock()
	defer p.Unlock()
	return map[string]float64{
		// XXX round up  here might be problem
		"sentBytes": float64(p.bytesSent),
		"rcvdBytes": float64(p.bytesRcvd),
	}
}
