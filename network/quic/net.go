package quic

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"sync"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
	quic "github.com/lucas-clemente/quic-go"
)

// Network is a handel.Network implementation using QUIC as its transport layer
type Network struct {
	mutex          sync.RWMutex
	listeners      []h.Listener
	quit           bool
	enc            network.Encoding
	quicListener   quic.Listener
	sessionManager sessionManager
}

const handshakeTimeout = 2000 * time.Millisecond

// NewNetwork creates Nework baked by QUIC protocol
func NewNetwork(addr string, enc network.Encoding) (*Network, error) {
	cfg := generateTLSConfig()
	qCfg := &quic.Config{HandshakeTimeout: handshakeTimeout} //, AcceptCookie: f}
	listener, err := quic.ListenAddr(addr, cfg, qCfg)

	if err != nil {
		panic(err)
	}
	var listeners []h.Listener
	sessManager := newSessionManager(handshakeTimeout)
	net := Network{
		mutex:          sync.RWMutex{},
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
	quicNet.mutex.Lock()
	defer quicNet.mutex.Unlock()
	quicNet.listeners = append(quicNet.listeners, listener)
}

// Stop stops the network
func (quicNet *Network) Stop() {
	quicNet.mutex.Lock()
	defer quicNet.mutex.Unlock()
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
		quicNet.mutex.RLock()
		quit := quicNet.quit
		listeners := quicNet.listeners
		enc := quicNet.enc
		quicNet.mutex.RUnlock()

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

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}
