// Package bn256 allows to use Handel with the BLS signature scheme over the
// BN256 groups. It implements the relevant Handel interfaces: PublicKey,
// Secretkey and Signature. The BN256 implementations comes from the
// cloudflare/bn256 package, including the base points..
package bn256

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ConsenSys/handel"
	"github.com/cloudflare/bn256"
)

// Hash is the hash function used to hash the message prior to signing
var Hash = sha256.New

// G2Base is the base point specified for the G2 group. If one wants to use a
// different point, set this variable before using any public methods / structs
// of this package.
var G2Base *bn256.G2

func init() {

	G2Base = new(bn256.G2)
	exp := big.NewInt(1)
	G2Base.ScalarBaseMult(exp)
}

// Constructor implements the handel.Constructor interface
type Constructor struct {
}

// NewConstructor returns a handel.Constructor capable of creating empty BLS
// signature object and empty public keys.
func NewConstructor() *Constructor {
	return &Constructor{}
}

// Signature implements the handel.Constructor  interface
func (s *Constructor) Signature() handel.Signature {
	return new(SigBLS)
}

// PublicKey implements the handel.Constructor interface
func (s *Constructor) PublicKey() handel.PublicKey {
	return new(PublicKey)
}

// SecretKey implements the simul/lib/Constructor interface
func (s *Constructor) SecretKey() handel.SecretKey {
	return new(SecretKey)
}

// KeyPair implements the simul/lib/Constructor interface
func (s *Constructor) KeyPair(r io.Reader) (handel.SecretKey, handel.PublicKey) {
	secret, pub, err := NewKeyPair(r)
	if err != nil {
		// this method is only used in simulation code anyway
		panic(err)
	}
	return secret, pub
}

// PublicKey holds the public key information = point in G2
type PublicKey struct {
	p *bn256.G2
}

func (p *PublicKey) String() string {
	//return p.p.String()
	buff, _ := p.MarshalBinary()
	s := sha256.Sum256(buff)
	return hex.EncodeToString(s[:])
}

// VerifySignature checks the given BLS signature bls on the message m using the
// public key p by verifying that the equality e(H(m), X) == e(H(m), x*B2) ==
// e(x*H(m), B2) == e(S, B2) holds where e is the pairing operation and B2 is
// the base point from curve G2.
func (p *PublicKey) VerifySignature(msg []byte, sig handel.Signature) error {
	ms := sig.(*SigBLS)
	HM, err := hashedMessage(msg)
	if err != nil {
		return err
	}
	leftPair := bn256.Pair(HM, p.p).Marshal()
	rightPair := bn256.Pair(ms.e, G2Base).Marshal()
	if !bytes.Equal(leftPair, rightPair) {
		return errors.New("bn256: signature invalid")
	}
	return nil
}

// Combine implements the handel.PublicKey interface
func (p *PublicKey) Combine(pp handel.PublicKey) handel.PublicKey {
	if p.p == nil {
		return pp
	}
	p2 := pp.(*PublicKey)
	p3 := new(bn256.G2)
	p3.Add(p.p, p2.p)
	return &PublicKey{p3}
}

// MarshalBinary implements the simul/lib/PublicKey interface
func (p *PublicKey) MarshalBinary() ([]byte, error) {
	return p.p.Marshal(), nil
}

// UnmarshalBinary implements the simul/lib/PublicKey interface
func (p *PublicKey) UnmarshalBinary(buff []byte) error {
	p.p = new(bn256.G2)
	_, err := p.p.Unmarshal(buff)
	return err
}

// SecretKey holds the secret scalar and can return the corresponding public
// key. It can sign messages using the BLS signature scheme.
type SecretKey struct {
	s *big.Int
}

// NewKeyPair returns a new keypair generated from the given reader.
func NewKeyPair(reader io.Reader) (*SecretKey, *PublicKey, error) {
	if reader == nil {
		reader = rand.Reader
	}
	secret, public, err := bn256.RandomG2(reader)
	if err != nil {
		return nil, nil, err
	}
	return &SecretKey{
			s: secret,
		}, &PublicKey{
			p: public,
		}, nil
}

// Sign creates a BLS signature S = x * H(m) on a message m using the private
// key x. The signature S is a point on curve G1.
func (s *SecretKey) Sign(msg []byte, reader io.Reader) (handel.Signature, error) {
	hashed, err := hashedMessage(msg)
	if err != nil {
		return nil, err
	}
	p := new(bn256.G1)
	p = p.ScalarMult(hashed, s.s)
	return &SigBLS{p}, nil
}

// MarshalBinary implements the simul/lib/SecretKey interface
func (s *SecretKey) MarshalBinary() ([]byte, error) {
	return s.s.Bytes(), nil
}

// UnmarshalBinary implements the simul/lib/SecretKey interface
func (s *SecretKey) UnmarshalBinary(buff []byte) error {
	s.s = new(big.Int)
	s.s = s.s.SetBytes(buff)
	return nil
}

// SigBLS represents a BLS signature using the BN256 curves
type SigBLS struct {
	e *bn256.G1
}

// MarshalBinary implements the handel.Signature interface
func (m *SigBLS) MarshalBinary() ([]byte, error) {
	if m.e == nil {
		return nil, errors.New("bn256: multisig can't marshal if nil")
	}
	return m.e.Marshal(), nil
}

// UnmarshalBinary implements the handel.Signature interface
func (m *SigBLS) UnmarshalBinary(b []byte) error {
	m.e = new(bn256.G1)
	_, err := m.e.Unmarshal(b)
	if err != nil {
		return errors.New("bn256: multisig can't unmarshal: " + err.Error())
	}
	return nil
}

// Combine implements the handel.Signature interface
func (m *SigBLS) Combine(ms handel.Signature) handel.Signature {
	if m.e == nil {
		return ms
	}
	m2 := ms.(*SigBLS)
	res := new(bn256.G1)
	res.Add(m.e, m2.e)
	return &SigBLS{e: res}
}

func (m *SigBLS) String() string {
	return m.e.String()
}

// hashedMessage returns the message hashed to G1
// XXX: this should be fixed as to have a method that maps a message
// (potentially a digest) to a point WITHOUT knowing the corresponding scalar.
func hashedMessage(msg []byte) (*bn256.G1, error) {
	h := Hash()
	_, err := h.Write(msg)
	if err != nil {
		return nil, err
	}
	hashed := h.Sum(nil)
	src := hex.EncodeToString(hashed)
	k, err := bigFromBase16(src)
	if err != nil {
		return nil, err
	}
	k.Abs(k)
	k.Mod(k, bn256.Order)
	scalar := big.NewInt(0).Mod(k, bn256.Order)
	return new(bn256.G1).ScalarBaseMult(scalar), nil
}

func bigFromBase16(s string) (*big.Int, error) {
	n, b := new(big.Int).SetString(s, 16)
	if b == false {
		err := fmt.Sprintf("Provided string %s is not a hexadecimal number.", s)
		return nil, errors.New(err)
	}
	return n, nil
}
