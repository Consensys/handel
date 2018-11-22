package libp2p

import (
	"bufio"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"

	h "github.com/ConsenSys/handel"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	proto "github.com/libp2p/go-libp2p-protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
	msmux "github.com/multiformats/go-multistream"
)

type libP2PNet struct {
	listeners *[]h.Listener
	host      host.Host
}

const protocol = "/handel/1.0.0"
const addrType = multiaddr.P_IPFS
const addrTypeStr = "ipfs"

// NewNetwork returns instance of handel.Network
func NewNetwork(listenPort int, id int32) h.Network {
	return NewLibP2pNetwork(listenPort, id)
}

// NewLibP2pNetwork returns instance of handel.Network which uses libp2p/tcp
// as a backend
func NewLibP2pNetwork(listenPort int, id int32) *libP2PNet {
	host := newHost(listenPort, id)
	listeners := &[]h.Listener{}
	host.SetStreamHandler(protocol, packetHandler(listeners))
	return &libP2PNet{listeners: listeners, host: host}
}

// RegisterListener appends Listener to the libP2PNet.listeners list,
// implements handel.Network interface
// This method is NOT thread safe
func (net *libP2PNet) RegisterListener(listener h.Listener) {
	*net.listeners = append(*net.listeners, listener)
}

// Send sends the given packet to the Identity using libp2p tcp transport
// implements handel.Network interface
func (net *libP2PNet) Send(identities []h.Identity, packet *h.Packet) {
	for _, id := range identities {
		net.send(id, packet)
	}
}

func (net *libP2PNet) send(identity h.Identity, packet *h.Packet) {
	peerid, multiAddr, err := makePeerIDAndAddr(identity.Address())
	if err != nil {
		fmt.Println("Error: unable to create remote peerid", err)
		return
	}

	//TODO set peerid, multiAddr only if Peerstore doesn't contain these entries
	net.host.Peerstore().SetAddr(*peerid, *multiAddr, pstore.PermanentAddrTTL)

	//	stream, err := net.host.NewStream(context.Background(), *peerid, protocol)
	s, err := net.newStream2(*peerid)

	if err != nil {
		log.Println("Error: can't establish connection to remote peer", identity.ID(), err)
		return
	}

	byteWriter := bufio.NewWriter(s)
	enc := gob.NewEncoder(byteWriter)
	err = enc.Encode(packet)
	if err != nil {
		log.Println("Error: unable to encode handel packet", err)
	}
	byteWriter.Flush()
	s.Close()
}

func (net *libP2PNet) newStream(peerid peer.ID) (net.Stream, error) {
	var protoStrs []string
	protoStrs = append(protoStrs, string(protocol))

	s, err := net.host.Network().NewStream(context.Background(), peerid)
	if err != nil {
		return nil, err
	}

	selected, err := msmux.SelectOneOf(protoStrs, s)
	if err != nil {
		s.Reset()
		return nil, err
	}
	selpid := proto.ID(selected)
	s.SetProtocol(selpid)
	net.host.Peerstore().AddProtocols(peerid, selected)
	return s, nil
}

func (net *libP2PNet) newStream2(peerid peer.ID) (net.Stream, error) {
	return net.host.NewStream(context.Background(), peerid, protocol)
}

// makeOpts creates default options for a host.
// 1- NAT traversal is enabled
// 2- libP2p identity is set in determinitic way, see MakeDeterministicID doc
func makeOpts(ip string, listenPort int, id int32) []libp2p.Option {
	prvKey, err := MakeDeterministicID(id)
	if err != nil {
		panic(err)
	}
	addr := fmt.Sprintf("/ip4/%s/tcp/%d", ip, listenPort)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(addr),
		libp2p.NATPortMap(),
		libp2p.Identity(prvKey),
	}
	return opts
}

// hostMultiAddr creates host multiaddress
// example:
// hostID - hex58 encoded libp2p peer.ID, QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX
// addr - ip4/127.0.0.1/tcp/3000
// returns ip4/127.0.0.1/tcp/3000/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX
func hostMultiAddr(hostID string, addr multiaddr.Multiaddr) multiaddr.Multiaddr {
	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/%s/%s", addrTypeStr, hostID))
	if err != nil {
		panic(err)
	}
	fullAddr := addr.Encapsulate(hostAddr)
	return fullAddr
}

func newHost(listenPort int, id int32) host.Host {
	opts := makeOpts("0.0.0.0", listenPort, id)
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Println("ERROR: can't create libP2P host")
		panic(err)
	}
	return host
}

// Returns the listen addresses of the Host
func (net *libP2PNet) HostMultiAddr() []multiaddr.Multiaddr {
	return net.host.Addrs()
}

func packetHandler(listeners *[]h.Listener) func(s net.Stream) {
	return func(s net.Stream) {
		var byteReader io.Reader = bufio.NewReader(s)
		var packet h.Packet
		dec := gob.NewDecoder(byteReader)
		err := dec.Decode(&packet)
		if err != nil {
			log.Println(err)
		}
		for _, listener := range *listeners {
			listener.NewPacket(&packet)
		}
		s.Close()
	}
}
