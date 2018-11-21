package main

import (
	"flag"
	"fmt"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network/libp2p"
	"github.com/ConsenSys/handel/network/libp2p/examples/localhost"
)

type ExampleListener struct {
	net h.Network
	reg h.Registry
	id  int32
}

func (l ExampleListener) NewPacket(packet *h.Packet) error {
	lvl := packet.Level
	fmt.Println("Lvl", lvl, "Org", packet.Origin, string(packet.MultiSig[:]))
	id, exsist := l.reg.Identity(int(packet.Origin))
	if !exsist {
		fmt.Println("don't exsist", packet.Origin)

	}

	newPacket := h.Packet{Origin: l.id, Level: lvl + 1, MultiSig: []byte("hello")}

	if l.id != id.ID() {
		ids := []h.Identity{id}
		l.net.Send(ids, &newPacket)
	}

	return nil
}

func main() {

	lPID := flag.Int("id", -1, "Peer id")
	regSize := flag.Int("reg", -1, "Registry size")
	flag.Parse()
	localPeerID := int32(*lPID)
	registrySize := int32(*regSize)
	port := 3000 + localPeerID

	net := libp2p.NewNetwork(int(port), localPeerID)
	localReg := localhost.NewLocalStaticRegistry(registrySize)
	net.RegisterListener(ExampleListener{net, localReg, localPeerID})
	packet := h.Packet{Origin: localPeerID, Level: 2, MultiSig: []byte("hello")}
	rids, _ := localReg.Identities(0, localReg.Size())
	rids = append(rids[:localPeerID], rids[localPeerID+1:]...)
	net.Send(rids, &packet)

	select {}
}
