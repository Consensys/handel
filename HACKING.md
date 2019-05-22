# Handel Architecture

Handel is meant to be a modularizable library which works over any network and
with any curves as long as they fulfill some properties. This document
highlights the general architecture of Handel and its key interfaces.

# Network

Handel must be able to work over a variety of network and be compatible with
any predefined wire-level network protocol. For example, it is possible to use
Handel over TCP, UDP,or inside predefined structure such as protobuf messages.

Handel interfaces with this generic network through the `Network` interface:
```go
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
```
As you can see, Handel only needs to know how to send `Packet`s and how to get
incoming `Packet`s. Handel's main structure `Handel` implements the `Listener`
interface.

# Identities 

Handel represents a participant,i.e. a signer, in the protocol thanks to the
`Identity` interface:
```go
// Identity holds the public information of a Handel node
type Identity interface {
	// Address must be understandable by the Network implementation
	Address() string
	// PublicKey returns the public key associated with that given node
	PublicKey() PublicKey
	// ID returns the ID used by handel to denote and classify nodes. It is best
	// if the IDs are continuous over a given finite range.
	ID() int32
}
```
Handel must know all `Identity` in order to be able to work correctly and
efficiently. Hence, Handel needs a `Registry` to easily access any `Identity`'s
public key:
```go
// Registry abstracts the bookeeping of the list of Handel nodes
type Registry interface {
	// Size returns the total number of Handel nodes
	Size() int
	// Identity returns the identity at this index in the registry, or
	// (nil,false) if the index is out of bound.
	Identity(int) (Identity, bool)
	// Identities is similar to Identity but returns an array instead that
	// includes nodes whose IDs are between from inclusive and to exclusive.
	Identities(from, to int) ([]Identity, bool)
}
```

# Cryptographic Keys & Signatures

Handel can be used to create multi-signature over any signature scheme
supporting aggregation. Handel represents the signatures and keys in a generic
way thanks to the following interfaces:
```go
// PublicKey represents either a generic individual or aggregate public key. It
// contain methods to verify a signature and to combine multiple public
// keys together to verify signatures.
type PublicKey interface {
	// VerifySignature takes a message and a signature and returns an error iif
	// the signature is invalid with respect to this public key and the message.
	VerifySignature(msg []byte, sig Signature) error
    // Combine combines the two public keys together to produce an aggregate
	// public key. The resulting public key must be valid and able to verify
	// aggregated signatures valid under the aggregate public key.
	Combine(PublicKey) PublicKey

	// String returns an easy representation of the public key (hex, etc).
	String() string
}

// SecretKey represents a secret key. 
type SecretKey interface {
	// Sign the given message using the given randomness source.
	Sign(msg []byte, r io.Reader) (Signature, error)
}

// Constructor creates empty signatures of the required type suitable for
// unmarshalling and empty public keys of the required type suitable for
// aggregation. See package bn256 for an example.
type Constructor interface {
	// Signature returns a fresh empty signature suitable for unmarshaling
	Signature() Signature
	// PublicKey returns a fresh empty public key suitable for aggregation
	PublicKey() PublicKey
}


// Signature holds methods to pass from/to a binary representation and to
// combine signatures together
type Signature interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error

	// Combine aggregates the two signature together producing an unique
	// signature that can be verified by the combination of both
	// respective public keys that produced the original signatures.
	Combine(Signature) Signature
}

// MultiSignature represents an aggregated signature alongside with its bitset.
// The signature is the aggregation of all individual signatures from the nodes
// whose index is set in the bitset.
type MultiSignature struct {
	BitSet
	Signature
}
```
As an example, you can see the implementation of these interfaces using BN256
curves in the `bn256` package.

**NOTE**: The `Constructor` interface is only useful to be able to
automatically unmarshal signatures from any incoming network's messages.
