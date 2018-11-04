package handel

// MultiSignature holds methods to pass from/to a binary representation and to
// combine multi-signatures together
type MultiSignature interface {
	MarshalBinary() ([]byte, error)

	// Combine "merges" the two signature together so that it produces an unique
	// multi-signature that can be verified by the combination of both
	// respective public keys that produced the original signatures.
	Combine(MultiSignature) MultiSignature
}

// we need
// +one method to create empty multisignatures, to unmarshal them into
// objects when receiving
// + one method to marshal
