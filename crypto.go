package handel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

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

// SecretKey represents a secret key. This interface is mostly needed to run the
// tests in a generic way
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

// Unmarshal reads a multisignature from the given slice, using the *empty*
// signature and bitset interface given.
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
	return fmt.Sprintf("{bitset %d/%d}",
		m.BitSet.Cardinality(), m.BitSet.BitLength())
}

// VerifyMultiSignature verifies a multisignature against the given message, aby
// aggregating all public keys from the registry. It returns nil if the
// verification was sucessful, an error otherwise.
func VerifyMultiSignature(msg []byte, ms *MultiSignature, reg Registry, cons Constructor) error {
	n := ms.BitSet.BitLength()
	if n != reg.Size() {
		return errors.New("verify multisignature: inconsistent sizes")
	}
	aggregate := cons.PublicKey()
	for i := 0; i < n; i++ {
		if ms.BitSet.Get(i) {
			id, ok := reg.Identity(i)
			if !ok {
				return fmt.Errorf("registry returned empty identity at %d", i)
			}
			aggregate = aggregate.Combine(id.PublicKey())
		}
	}

	return aggregate.VerifySignature(msg, ms.Signature)
}
