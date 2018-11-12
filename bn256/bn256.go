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
	"io"
	"math/big"

	"github.com/ConsenSys/handel"
	"github.com/cloudflare/bn256"
)

// G1Str is the hexadecimal string representing the base specified for the G1
// base point. It is taken from the cloudfare's bn256 implementation.
var G1Str = "00000000000000000000000000000000000000000000000000000000000000018fb501e34aa387f9aa6fecb86184dc21ee5b88d120b5b59e185cac6c5e089665"

// G1Base is the base point specified for the G1 group. If one wants to use a
// different point, set this variable before using any public methods / structs
// of this package.
var G1Base *bn256.G1

// G2Str is the hexadecimal string representing the base specified for the G1
// base point.
var G2Str = "012ecca446ff6f3d4d03c76e9b5c752f28bc37b364cb05ac4a37eb32e1c32459708f25386f72c9462b81597d65ae2092c4b97792155dcdaad32b8a6dd41792534c2db10ef5233b0fe3962b9ee6a4bbc2b5bde01a54f3513d42df972e128f31bf12274e5747e8cafacc3716cc8699db79b22f0e4ff3c23e898f694420a3be3087a5"

// G2Base is the base point specified for the G2 group. If one wants to use a
// different point, set this variable before using any public methods / structs
// of this package.
var G2Base *bn256.G2

// Hash is the hash function used to digest a message before mapping it to a
// point.
var Hash = sha256.New

func init() {
	buff, err := hex.DecodeString(G1Str)
	if err != nil {
		panic("bn256: can't decode base point on G1. Fatal error")
	}
	G1Base = new(bn256.G1)
	_, err = G1Base.Unmarshal(buff)
	if err != nil {
		panic("bn256: can't decode base point on G1. Fatal error")
	}

	buff, err = hex.DecodeString(G2Str)
	if err != nil {
		panic("bn256: can't decode base point on G2. Fatal error.")
	}
	G2Base = new(bn256.G2)
	_, err = G2Base.Unmarshal(buff)
	if err != nil {
		panic("bn256: can't decode base point on G2. Fatal error.")
	}
}

// scheme implements the handel.SignatureScheme interface
type scheme struct {
	handel.SecretKey
}

// NewSignatureScheme returns a signature scheme using the provided secret key
// for the bn256 BLS multi-signatures.
func NewSignatureScheme(s handel.SecretKey) handel.SignatureScheme {
	return &scheme{s}
}

func (s *scheme) UnmarshalSignature(buff []byte) (handel.Signature, error) {
	ms := new(bls)
	return ms, ms.UnmarshalBinary(buff)
}

type publicKey struct {
	p *bn256.G2
}

func (p *publicKey) String() string {
	return p.p.String()
}

// VerifySignature checks the given BLS signature bls on the message m using the
// public key p by verifying that the equality e(H(m), X) == e(H(m), x*B2) ==
// e(x*H(m), B2) == e(S, B2) holds where e is the pairing operation and B2 is
// the base point from curve G2.
func (p *publicKey) VerifySignature(msg []byte, sig handel.Signature) error {
	ms := sig.(*bls)
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

func (p *publicKey) Combine(pp handel.PublicKey) handel.PublicKey {
	p2 := pp.(*publicKey)
	p3 := new(bn256.G2)
	p3.Add(p.p, p2.p)
	return &publicKey{p3}
}

type secretKey struct {
	*publicKey
	s *big.Int
}

// NewSecretKey returns a new keypair generated from the given reader.
func NewSecretKey(reader io.Reader) (handel.SecretKey, error) {
	if reader == nil {
		reader = rand.Reader
	}
	secret, public, err := bn256.RandomG2(reader)
	if err != nil {
		return nil, err
	}
	return &secretKey{
		s: secret,
		publicKey: &publicKey{
			p: public,
		},
	}, nil
}

func (s *secretKey) PublicKey() handel.PublicKey {
	return s.publicKey
}

// Sign creates a BLS signature S = x * H(m) on a message m using the private
// key x. The signature S is a point on curve G1.
func (s *secretKey) Sign(msg []byte, reader io.Reader) (handel.Signature, error) {
	hashed, err := hashedMessage(msg)
	if err != nil {
		return nil, err
	}
	p := new(bn256.G1)
	p = p.ScalarMult(hashed, s.s)
	return &bls{p}, nil
}

type bls struct {
	e *bn256.G1
}

func (m *bls) MarshalBinary() ([]byte, error) {
	if m.e == nil {
		return nil, errors.New("bn256: multisig can't marshal if nil")
	}
	return m.e.Marshal(), nil
}

func (m *bls) UnmarshalBinary(b []byte) error {
	m.e = new(bn256.G1)
	_, err := m.e.Unmarshal(b)
	if err != nil {
		return errors.New("bn256: multisig can't unmarshal: " + err.Error())
	}
	return nil
}

func (m *bls) Combine(ms handel.Signature) handel.Signature {
	m2 := ms.(*bls)
	res := new(bn256.G1)
	res.Add(m.e, m2.e)
	return &bls{e: res}
}

// hashedMessage returns the message hashed to G1
// XXX: this should be fixed as to have a method that maps a message
// (potentially a digest) to a point WITHOUT knowing the corresponding scalar.
func hashedMessage(msg []byte) (*bn256.G1, error) {
	h := Hash()
	h.Write(msg)
	hashed := h.Sum(nil)
	reader := bytes.NewBuffer(hashed)
	_, HM, err := bn256.RandomG1(reader)
	return HM, err

}
