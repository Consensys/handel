package handel

// Network is the interface that must be given to Handel to communicate with
// other Handel instances. A Network implementation does not need to provide any
// transport layer guarantees (such as delivery or in-order).
type Network interface {
	// RegisterListener stores a Listener to dispatch incoming messages to it
	// later on
	RegisterListener(Listener)
	// Send sends the given packet to the given Identity. There can be no
	// guarantees about the reception of the packet provided by the Network.
	Send([]Identity, *Packet)
}

// Listener is the interface that gets registered to the Network. Each time a
// new packet arrives from the network, it is dispatched to the registered
// Listeners.
type Listener interface {
	NewPacket(*Packet)
}

// ListenFunc is a wrapper type to morph a function as a Listener
type ListenFunc func(*Packet)

// NewPacket implements the Listener interface
func (l ListenFunc) NewPacket(p *Packet) {
	l(p)
}

// Packet is the general packet that Handel sends out and expects to receive
// from the Network. Handel do not provide any authentication nor
// confidentiality on Packets, it is up to the application layer to add these
// features if relevant.
type Packet struct {
	// Origin is the ID of the sender of this packet.
	Origin int32
	// Level indicates for which level this packet is for in the Handel tree.
	// Values start at 1. There is no level 0.
	Level byte
	// MultiSig holds a MultiSignature struct.
	MultiSig []byte
	// IndividualSig holds the individual signature of the Origin node
	IndividualSig []byte
}
