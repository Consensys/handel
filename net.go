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
	NewPacket(ApplicationPacket)
}

// ListenFunc is a wrapper type to be able to register a function as a Listener
type ListenFunc func(ApplicationPacket)

// NewPacket implements the Listener interface
func (l ListenFunc) NewPacket(p ApplicationPacket) {
	l(p)
}

// ApplicationPacket
type ApplicationPacket interface {
	Handel() *Packet
	ID() int32
}

type defaultApplicationPacket struct {
	packet *Packet
}

func NewAppPacket(packet *Packet) ApplicationPacket {
	return &defaultApplicationPacket{packet}
}

func (ap *defaultApplicationPacket) Handel() *Packet {
	return ap.packet
}

func (ap *defaultApplicationPacket) ID() int32 {
	return ap.Handel().Origin
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

/*// MarshalBinary implements the go BinaryMarshaler interface*/
//func (p *Packet) MarshalBinary() ([]byte, error) {
//var buffer bytes.Buffer
//binary.Write(&buffer, binary.BigEndian, p.Origin)
//binary.Write(&buffer, binary.BigEndian, p.Level)
//buffer.Write(p.MultiSig)
//buffer.Write(p.IndividualSig)
//return buffer.Bytes(), nil
/*}*/

// UnmarshalBinary implements the go BinaryUnmarshaler interface
/*func (p *Packet) UnmarshalBinary(buff []byte) error {*/
//var buffer = bytes.NewBuffer(buff)
//err := binary.Read(buffer, binary.BigEndian, &p.Origin)
//if err != nil {
//return err
//}
//err = binary.Read(buffer, binary.BigEndian, &p.Level)
//if err != nil {
//return err
//}
//p.MultiSig = buffer.Bytes()
//return nil
/*}*/
