package lib

import (
	"io"

	"github.com/ConsenSys/handel"
)

// Marshallable represents an interface that can marshal and unmarshals itself
type Marshallable interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(buff []byte) error
}

// Constructor can construct a secret key on top of the regular handel
// constructor. This is what NEW CURVES must implement in order to be tested on
// a simulation.
type Constructor interface {
	Handel() handel.Constructor
	PublicKey() PublicKey
	SecretKey() SecretKey
	Signature() handel.Signature
	KeyPair(r io.Reader) (SecretKey, PublicKey)
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

// lots of machinery here to be able to not put explicit additional constraints
// of new curves for Handel compatibility. We should not have to look at it :)

// SecretConstructor is an interface that can issue a secret key. Useful since
// SecretKey() method is not mandatory in Handel.
type SecretConstructor interface {
	SecretKey() handel.SecretKey
}

// Generator can generate a new key pair
type Generator interface {
	KeyPair(io.Reader) (handel.SecretKey, handel.PublicKey)
}

type handelConstructor struct {
	c handel.Constructor
}

// NewSimulConstructor returns a simulation Constructor
func NewSimulConstructor(h handel.Constructor) Constructor {
	return &handelConstructor{h}
}

func (h *handelConstructor) PublicKey() PublicKey {
	return h.c.PublicKey().(PublicKey)
}
func (h *handelConstructor) SecretKey() SecretKey {
	return h.c.(SecretConstructor).SecretKey().(SecretKey)
}
func (h *handelConstructor) Signature() handel.Signature {
	return h.c.Signature()
}
func (h *handelConstructor) Handel() handel.Constructor {
	return h.c
}
func (h *handelConstructor) KeyPair(r io.Reader) (SecretKey, PublicKey) {
	sec, pub := h.c.(Generator).KeyPair(r)
	return sec.(SecretKey), pub.(PublicKey)
}

type emptyConstructor struct{}

// NewEmptyConstructor returns a Constructor that construct fake secret key and
// fake public key that don't do anything. Useful for testing networks and the
// likes. Do NOT use the public / secret / signatures for anything !
func NewEmptyConstructor() Constructor {
	return new(emptyConstructor)
}

func (e *emptyConstructor) Signature() handel.Signature {
	return new(fakeSig)
}

func (e *emptyConstructor) SecretKey() SecretKey {
	return new(fakeSecret)
}

func (e *emptyConstructor) PublicKey() PublicKey {
	return new(fakePublic)
}

func (e *emptyConstructor) Handel() handel.Constructor {
	return nil
}

func (e *emptyConstructor) KeyPair(r io.Reader) (SecretKey, PublicKey) {
	return nil, nil
}

type fakePublic struct{}

func (f *fakePublic) String() string {
	return ""
}
func (f *fakePublic) VerifySignature(b []byte, s handel.Signature) error {
	return nil
}

func (f *fakePublic) Combine(p handel.PublicKey) handel.PublicKey {
	return f
}

func (f *fakePublic) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

func (f *fakePublic) UnmarshalBinary(b []byte) error {
	return nil
}

type fakeSecret struct{}

func (f *fakeSecret) Sign(msg []byte, rand io.Reader) (handel.Signature, error) {
	return &fakeSig{}, nil
}

func (f *fakeSecret) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

func (f *fakeSecret) UnmarshalBinary(b []byte) error {
	return nil
}

type fakeSig struct{}

func (f *fakeSig) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

func (f *fakeSig) UnmarshalBinary(buff []byte) error {
	return nil
}

func (f *fakeSig) Combine(handel.Signature) handel.Signature {
	return f
}

func (f *fakeSig) String() string {
	return ""
}
