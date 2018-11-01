package handel

// Network is the interface that must be given to Handel to communicate with
// other Handel instances. A Network implementation does not need to provide any
// transport layer guarantees (such as delivery or in-order).
type Network interface {
	// RegisterListener stores a Listener to dispatch incoming messages to it
	// later on. Implementations must allow multiple Listener to be registered.
	RegisterListener(Listener)
	// Send sends the given packet to the given Identity. There can be no
	// guarantees about the reception of the packet provided by the Network.
	Send(Identity, *Packet) error
}

// Listener is the interface that gets registered to the Network. Each time a
// new packet arrives from the network, it is dispatched to the registered
// Listeners.
type Listener interface {
	NewPacket(*Packet)
}

// Packet is the general packet that Handel sends out and expects to receive
// from the Network.
type Packet struct {
	Origin int
	Level  int
	Sig    MultiSignature
}

// MarshalBinary implements the go BinaryMarshaler interface
func (p *Packet) MarshalBinary() ([]byte, error) {
	panic("not implemented yet")
}
