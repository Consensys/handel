package quic

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"sync"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	quic "github.com/lucas-clemente/quic-go"
)

// Network is a handel.Network implementation using QUIC as its transport layer
type Network struct {
	sync.RWMutex
	listeners      []h.Listener
	quit           bool
	enc            network.Encoding
	quicListener   quic.Listener
	sessionManager sessionManager
}

// NewNetwork creates Nework baked by QUIC protocol
func NewNetwork(addr string, enc network.Encoding, cfg Config) (*Network, error) {
	//	cfg := cfg. generateTLSConfig()
	qCfg := &quic.Config{HandshakeTimeout: cfg.handshakeTimeout} //, AcceptCookie: f}
	listener, err := quic.ListenAddr(addr, cfg.tlsCfg, qCfg)

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

func (quicNet *Network) handler() {
	for {
		sess, err := quicNet.quicListener.Accept()

		if err != nil {
			panic(err)
		}
		quicNet.RLock()
		quit := quicNet.quit
		listeners := quicNet.listeners
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
	reader := bufio.NewReader(stream)
	dispatch(listeners, reader, enc)
	io.Copy(ioutil.Discard, stream)
	stream.Close()
	sess.Close()
}

func dispatch(listeners []h.Listener, byteReader io.Reader, enc network.Encoding) {
	packet, err := enc.Decode(byteReader)

	if err != nil {
		log.Println(err)
	}
	for _, listener := range listeners {
		listener.NewPacket(packet)
	}
}
