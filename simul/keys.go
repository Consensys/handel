package simul

import (
	"github.com/ConsenSys/handel"
)

// Marshallable represents an interface that can marshal and unmarshals itself
type Marshallable interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(buff []byte) error
}

// Constructor can construct a secret key on top of the regular handel
// constructor
type Constructor interface {
	handel.Constructor
	SecretKey() SecretKey
}

// SecretKey can also Marshal itself on top of the regular handel SecretKey
type SecretKey interface {
	Marshallable
	handel.SecretKey
}

// PublicKey can also Marshal itself on top of the regular handel PublicKey
type PublicKey interface {
	handel.PublicKey
	Marshallable
}
