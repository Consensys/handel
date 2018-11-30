package handel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

// SecretKey is an înterface holding the required functionality of a secret key
// needed to run the generic tests.
type SecretKey interface {
	Sign(msg []byte, r io.Reader) (Signature, error)
}

// Constructor is used to create empty signatures suitable for unmarshalling and
// empty public key suitable for aggregation.
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

	// Combine "merges" the two signature together so that it produces an unique
	// multi-signature that can be verified by the combination of both
	// respective public keys that produced the original signatures.
	Combine(Signature) Signature
}

// MultiSignature represents an aggregated signature alongside with its bitset.
// Handel outputs potentially multiple MultiSignatures during the protocol.
// The BitSet is always expected to have the maximum size.
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
func (m *MultiSignature) Unmarshal(b []byte, s Signature, nbs func(b int) BitSet) error {
	var buff = bytes.NewBuffer(b)
	var length uint16
	if err := binary.Read(buff, binary.BigEndian, &length); err != nil {
		return err
	}

	bitset := buff.Next(int(length))
	if len(bitset) < int(length) {
		return errors.New("bitset received smaller than expected")
	}

	bs := nbs(int(length))
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

func (m *MultiSignature) String() string {
	return fmt.Sprintf("{bs (len %d - card %d): %s, ms: %s}",
		m.BitSet.BitLength(), m.BitSet.Cardinality(), m.BitSet, m.Signature)
}
