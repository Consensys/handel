package tcp

import (
	"net"
	"sync"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network"
)

// value given to SetDeadLine on all connections - TTL equivalent
var timeout = 1 * time.Minute

// Network implements the handel.Network interface using TCP connections
type Network struct {
	sync.Mutex
	addr     string
	l        net.Listener
	conns    map[string]net.Conn
	enc      network.Encoding
	listener h.Listener
}

// NewNetwork returns a TCP Network that listens to the given address.
func NewNetwork(listen string, enc network.Encoding) (*Network, error) {
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}
	n := &Network{
		addr:  listen,
		l:     listener,
		enc:   enc,
		conns: make(map[string]net.Conn),
	}
	go n.handleIncoming()
	return n, nil
}

func (n *Network) handleIncoming() {
	for {
		conn, err := n.l.Accept()
		if err != nil {
			return
		}
		n.registerConn(conn)
		go n.handleConn(conn)
	}
}

func (n *Network) handleConn(c net.Conn) {
	for {
		c.SetDeadline(time.Now().Add(timeout))
		//reader := bufio.NewReader(c)
		//packet, err := n.enc.Decode(reader)
		packet, err := n.enc.Decode(c)
		if err != nil {
			return
		}
		n.dispatch(packet)
	}
}

func (n *Network) registerConn(c net.Conn) {
	n.Lock()
	defer n.Unlock()
	n.conns[c.RemoteAddr().String()] = c
}

func (n *Network) unregisterConn(c net.Conn) {
	n.Lock()
	defer n.Unlock()
	delete(n.conns, c.RemoteAddr().String())
}

// Send implements the handel.Network interface
func (n *Network) Send(ids []h.Identity, packet *h.Packet) {
	n.Lock()
	defer n.Unlock()
	var err error
	for _, id := range ids {
		addr := id.Address()
		conn, exists := n.conns[addr]
		if !exists {
			conn, err = n.connectTo(addr)
		}
		//byteWriter := bufio.NewWriter(conn)
		if err = n.enc.Encode(packet, conn); err != nil {
			go n.unregisterConn(conn)
			continue
		}
	}
}

func (n *Network) connectTo(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	n.conns[addr] = conn
	go n.handleConn(conn)
	return conn, nil
}

// Stop the listener
func (n *Network) Stop() {
	n.Lock()
	defer n.Unlock()
	n.l.Close()
	for _, c := range n.conns {
		c.Close()
	}
}

// RegisterListener implements the h.Network interface
func (n *Network) RegisterListener(listener h.Listener) {
	n.Lock()
	defer n.Unlock()
	n.listener = listener
}

func (n *Network) dispatch(p *h.Packet) {
	n.Lock()
	defer n.Unlock()
	n.listener.NewPacket(h.NewAppPacket(p))
}
