package localhost

import (
	"flag"
	"fmt"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/network/libp2p"
)

type ExampleListener struct {
	net h.Network
	reg h.Registry
	id  int32
}

func (l ExampleListener) NewPacket(packet *h.Packet) error {
	lvl := packet.Level
	fmt.Println("Recived msg ", "Lvl", lvl, "Org", packet.Origin, string(packet.MultiSig[:]))

	sig := "Hello" //string(packet.MultiSig) + " signed by " + strconv.Itoa(int(l.id)) + ", "

	newPacket := h.Packet{Origin: l.id, Level: lvl + 1, MultiSig: []byte(sig)}
	rids, _ := l.reg.Identities(0, l.reg.Size())

	//Send packet to all peers except from myself
	tmp := removePeer(rids, int(l.id))
	l.net.Send(tmp, &newPacket)

	return nil
}

func Start() {
	lPID := flag.Int("id", -1, "Peer id")
	regSize := flag.Int("reg", -1, "Registry size")
	flag.Parse()
	localPeerID := int32(*lPID)
	registrySize := int32(*regSize)
	port := 3000 + localPeerID

	net := libp2p.NewLibP2pNetwork(int(port), localPeerID)

	// adresses[0] - localhost adress
	// adresses[1] - localnetwork adress
	// adresses[2] - external adress, if nat travesal enabled
	// we need to wait some period of time for libp2p to ask router about host
	// external adress
	time.Sleep(3 * time.Second)
	fmt.Println("Host listen adresses")
	for _, addr := range net.HostMultiAddr() {
		fmt.Println(addr)
	}
	localReg := NewLocalStaticRegistry(registrySize)
	net.RegisterListener(ExampleListener{net, localReg, localPeerID})

	packet := h.Packet{Origin: localPeerID, Level: 0, MultiSig: []byte("hello")}
	rids, _ := localReg.Identities(0, localReg.Size())

	//Send packet to all peers except from myself
	tmp := removePeer(rids, int(localPeerID))
	net.Send(tmp, &packet)
	select {}
}

func removePeer(rids []h.Identity, localPeerID int) []h.Identity {
	tmp := make([]h.Identity, len(rids))
	copy(tmp, rids)
	tmp = append(tmp[:localPeerID], tmp[localPeerID+1:]...)
	return tmp
}
