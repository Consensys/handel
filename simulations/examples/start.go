package main

import (
	"flag"
	"fmt"

	h "github.com/ConsenSys/handel"
	network "github.com/ConsenSys/handel/network"
	udp "github.com/ConsenSys/handel/network/udp"

	"github.com/ConsenSys/handel/simulations/utils"
)

type exampleListener struct {
	net h.Network
	reg h.Registry
	id  int32
}

//NewPacket
func (l exampleListener) NewPacket(packet *h.Packet) {
	lvl := packet.Level
	bs := make([]byte, 1200)

	fmt.Println("msg received:", "Lvl", lvl, "Org", packet.Origin, len(packet.MultiSig))

	newPacket := h.Packet{Origin: l.id, Level: lvl + 1, MultiSig: bs}

	rids, _ := l.reg.Identities(0, l.reg.Size())

	//Send packet to all peers except from myself
	peers := removePeer(rids, int(l.id))
	l.net.Send(peers, &newPacket)
}

func start() {
	lPID := flag.Int("id", -1, "Peer id")
	reg := flag.String("reg", "", "Path to registry file")
	flag.Parse()
	localPeerID := int32(*lPID)

	lineParser := utils.NewEmptyPublicKeyCsvParser()

	registry, port := utils.ReadCSVRegistry(*reg, localPeerID, lineParser)

	enc := network.NewGOBEncoding()
	net := udp.NewUDPNetwork(port, enc)

	listener := exampleListener{net, registry, localPeerID}
	net.RegisterListener(listener)

	packet := h.Packet{Origin: localPeerID, Level: 0, MultiSig: []byte("hello")}

	rids, _ := registry.Identities(0, registry.Size())

	//Send packet to all peers except from myself
	peers := removePeer(rids, int(localPeerID))

	net.Send(peers, &packet)
	//time.Sleep(3 * time.Second)
	//net.Stop()

	select {}
}

func removePeer(rids []h.Identity, localPeerID int) []h.Identity {
	tmp := make([]h.Identity, len(rids))
	copy(tmp, rids)
	tmp = append(tmp[:localPeerID], tmp[localPeerID+1:]...)
	return tmp
}

func main() {
	start()
}
