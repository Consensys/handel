package handel

import "crypto/cipher"

// PublicKey holds methods to verify a signature and to combine multiple public
// keys together to verify multi-signatures.
type PublicKey interface {
	String() string
	VerifySignature([]byte) error
	// Combine combines two public keys together so that a multi-signature
	// produced by both individual public keys can be verified by the combined
	// public key
	Combine(PublicKey) PublicKey
}

// SecretKey holds methods to produce a valid signature that can be verified
// under the corresponding public key.
type SecretKey interface {
	PublicKey
	Sign(rand cipher.Stream) ([]byte, error)
}
