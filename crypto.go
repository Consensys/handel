package handel

import "io"

// PublicKey holds methods to verify a signature and to combine multiple public
// keys together to verify signatures.
type PublicKey interface {
	String() string
	VerifySignature(msg []byte, sig Signature) error
	// Combine combines two public keys together so that a multi-signature
	// produced by both individual public keys can be verified by the combined
	// public key
	Combine(PublicKey) PublicKey
}

// SecretKey holds methods to produce a valid signature that can be verified
// under the corresponding public key.
type SecretKey interface {
	PublicKey() PublicKey
	// Sign returns a signature over the given message and using the reader for
	// any randomness necessary, if any. The rand argument can be left nil.
	Sign(msg []byte, rand io.Reader) (Signature, error)
}

// SignatureScheme holds a private key interface and a method to unmarshal
// multisignatures
type SignatureScheme interface {
	SecretKey
	UnmarshalSignature([]byte) (Signature, error)
}

// Signature holds methods to pass from/to a binary representation and to
// combine signatures together
type Signature interface {
	MarshalBinary() ([]byte, error)

	// Combine "merges" the two signature together so that it produces an unique
	// multi-signature that can be verified by the combination of both
	// respective public keys that produced the original signatures.
	Combine(Signature) Signature
}

// MultiSignature represents an aggregated signature alongside with its bitset.
// Handel outputs potentially multiple MultiSignatures during the protocol.
type MultiSignature struct {
	BitSet
	Signature
}
