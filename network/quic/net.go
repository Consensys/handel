package quic

import (
	"bufio"
	"log"
	"net"
	"sync"

	"github.com/ConsenSys/handel"
	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	quic "github.com/lucas-clemente/quic-go"
)

// This implementation spawns new session per every packet.
// This simplifies the session managment part but on the other hand prevents
// us from benefiting from the 0-RTT feature of quic.
// TODO add another Network quic implementation with session caching which will
// allow to enable 0-RTT

// Network is a handel.Network implementation using QUIC as its transport layer
type Network struct {
	sync.RWMutex
	listeners      []h.Listener
	quit           bool
	enc            network.Encoding
	quicListener   quic.Listener
	sessionManager sessionManager
	sent           int
	rcvd           int
}

func AcceptCookie(clientAddr net.Addr, cookie *quic.Cookie) bool {
	return true
}

// NewNetwork creates Nework baked by QUIC protocol
func NewNetwork(addr string, enc network.Encoding, cfg Config) (*Network, error) {
	//	cfg := cfg. generateTLSConfig()
	qCfg := &quic.Config{HandshakeTimeout: cfg.handshakeTimeout, AcceptCookie: AcceptCookie}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	newAddr := net.JoinHostPort("0.0.0.0", port)
	listener, err := quic.ListenAddr(newAddr, cfg.tlsCfg, qCfg)

	if err != nil {
		panic(err)
	}
	var listeners []h.Listener
	sessManager := newSessionManager(cfg.dialer)
	net := Network{
		listeners:      listeners,
		quit:           false,
		enc:            enc,
		quicListener:   listener,
		sessionManager: sessManager,
	}

	go net.handler()
	return &net, nil
}

//RegisterListener registers listener for processing incoming packets
func (quicNet *Network) RegisterListener(listener h.Listener) {
	quicNet.Lock()
	defer quicNet.Unlock()
	quicNet.listeners = append(quicNet.listeners, listener)
}

// Stop stops the network
func (quicNet *Network) Stop() {
	quicNet.Lock()
	defer quicNet.Unlock()
	quicNet.quit = true
}

//Send sends a packet to supplied identities
func (quicNet *Network) Send(identities []h.Identity, packet *h.Packet) {
	quicNet.Lock()
	quicNet.sent += len(identities)
	quicNet.Unlock()

	for _, id := range identities {
		go quicNet.send(id, packet)
	}
}

func (quicNet *Network) send(identity h.Identity, packet *h.Packet) {
	dialResult := quicNet.sessionManager.Dial(identity)

	if dialResult.isWaiting || dialResult.err != nil {
		return
	}
	stream, err := dialResult.session.OpenStream()

	byteWriter := bufio.NewWriter(stream)
	quicNet.enc.Encode(packet, byteWriter)

	if err != nil {
		panic(err)
	}
	byteWriter.Flush()
	stream.Close()
}

func (quicNet *Network) getListeners() []handel.Listener {
	quicNet.RLock()
	defer quicNet.RUnlock()
	quicNet.rcvd++
	return quicNet.listeners
}

func (quicNet *Network) handler() {
	for {
		sess, err := quicNet.quicListener.Accept()

		if err != nil {
			panic(err)
		}
		quicNet.RLock()
		quit := quicNet.quit
		listeners := quicNet.getListeners()
		enc := quicNet.enc
		quicNet.RUnlock()

		if quit {
			sess.Close()
			return
		}
		go handleSession(sess, listeners, enc)
	}
}

func handleSession(sess quic.Session, listeners []h.Listener, enc network.Encoding) {
	stream, err := sess.AcceptStream()

	if err != nil {
		return
	}
	packet, err := enc.Decode(stream)
	stream.Close()
	sess.Close()

	if err != nil {
		log.Println(err)
	}
	for _, listener := range listeners {
		listener.NewPacket(h.NewAppPacket(packet))
	}
}

// Values implements the monitor.CounterMeasure interface
func (quicNet *Network) Values() map[string]float64 {
	quicNet.RLock()
	defer quicNet.RUnlock()
	toSend := map[string]float64{
		"sent": float64(quicNet.sent),
		"rcvd": float64(quicNet.rcvd),
	}
	counter, ok := quicNet.enc.(*network.CounterEncoding)
	if ok {
		for k, v := range counter.Values() {
			toSend[k] = v
		}
	}
	return toSend
}
