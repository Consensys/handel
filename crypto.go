package handel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

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

// SignatureScheme holds a private key interface and a method to create empty
// signatures suitable for unmarshalling
type SignatureScheme interface {
	SecretKey
	// Signature returns a fresh empty signature suitable for unmarshaling
	Signature() Signature
}

// Signature holds methods to pass from/to a binary representation and to
// combine signatures together
type Signature interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error

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

// MarshalBinary implements the binary.Marshaller interface
func (m *MultiSignature) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	bs, err := m.BitSet.MarshalBinary()
	if err != nil {
		return nil, err
	}
	length := uint16(len(bs))

	sig, err := m.Signature.MarshalBinary()
	if err != nil {
		return nil, err
	}

	binary.Write(&b, binary.BigEndian, length)
	b.Write(bs)
	b.Write(sig)
	return b.Bytes(), nil
}

// Unmarshal reads a multisignature from the given slice, using the signature
// and bitset interface given.
func (m *MultiSignature) Unmarshal(b []byte, s Signature, bs BitSet) error {
	var buff = bytes.NewBuffer(b)
	var length uint16
	if err := binary.Read(buff, binary.BigEndian, &length); err != nil {
		return err
	}

	bitset := buff.Next(int(length))
	if len(bitset) < int(length) {
		return errors.New("bitset received smaller than expected")
	}
	if err := bs.UnmarshalBinary(bitset); err != nil {
		return err
	}

	if err := s.UnmarshalBinary(buff.Bytes()); err != nil {
		return err
	}

	m.BitSet = bs
	m.Signature = s
	return nil
}
