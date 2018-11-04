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

type Listener interface {
	NewPacket(*Packet)
}

```
As you can see, Handel only needs to know how to send messages and how to get
incoming messages. Handel's main structure `Handel` implements the `Listener`
interface.

# Identities 

Handel represents a participant,i.e. a signer, in the protocol thanks to the
`Identity` interface:
```go
type Identity interface {
	Address() string
	// PublicKey returns the public key associated with that given node
	PublicKey() PublicKey
}
```
Handel must know all `Identity` in order to be able to work correctly and
efficiently. Hence, Handel needs a `Registry` to easily access any `Identity`'s
public key easily:
```go
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
type PublicKey interface {
	String() string
	VerifySignature(msg []byte, sig MultiSignature) error
	// Combine combines two public keys together so that a multi-signature
	// produced by both individual public keys can be verified by the combined
	// public key
	Combine(PublicKey) PublicKey
}

type SecretKey interface {
	PublicKey() PublicKey
	// Sign returns a signature over the given message and using the reader for
	// any randomness necessary, if any. The rand argument can be left nil.
	Sign(msg []byte, rand io.Reader) (MultiSignature, error)
}

type MultiSignature interface {
	MarshalBinary() ([]byte, error)

	// Combine "merges" the two signature together so that it produces an unique
	// multi-signature that can be verified by the combination of both
	// respective public keys that produced the original signatures.
	Combine(MultiSignature) MultiSignature
}

// SignatureScheme holds a private key interface and a method to unmarshal
// multisignatures
type SignatureScheme interface {
	SecretKey
	UnmarshalSignature([]byte) (MultiSignature, error)
}
```
As an example, you can see the implementation of these interfaces using BN256
curves in the `bn256` package.

**NOTE**: The `SignatureScheme` interface is only useful to be able to
automatically unmarshal signatures from any incoming network's messages.
