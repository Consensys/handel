package handel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
)

func FakeRegistry(size int) Registry {
	ids := make([]Identity, size)
	for i := 0; i < size; i++ {
		ids[i] = &fakeIdentity{int32(i), &fakePublic{true}}
	}
	return NewArrayRegistry(ids)
}

type fakePublic struct {
	verify bool
}

func (f *fakePublic) String() string {
	return fmt.Sprintf("public-%v", f.verify)
}
func (f *fakePublic) VerifySignature(b []byte, s Signature) error {
	if !f.verify {
		return errors.New("wrong")
	}
	if !s.(*fakeSig).verify {
		return errors.New("wrong")
	}
	return nil
}
func (f *fakePublic) Combine(p PublicKey) PublicKey {
	return &fakePublic{f.verify && p.(*fakePublic).verify}
}

type fakeIdentity struct {
	id int32
	*fakePublic
}

func (f *fakeIdentity) Address() string {
	return fmt.Sprintf("fake-%d-%v", f.id, f.fakePublic.verify)
}
func (f *fakeIdentity) PublicKey() PublicKey { return f.fakePublic }
func (f *fakeIdentity) ID() int32            { return f.id }
func (f *fakeIdentity) String() string       { return f.Address() }

type fakeSecret struct {
}

func (f *fakeSecret) Public() PublicKey {
	return new(fakePublic)
}

func (f *fakeSecret) Sign(msg []byte, rand io.Reader) (Signature, error) {
	return &fakeSig{}, nil
}

var fakeConstSig = []byte{0x01, 0x02, 0x3, 0x04}

type fakeSig struct {
	card   int // cardinality
	verify bool
}

func (f *fakeSig) MarshalBinary() ([]byte, error) {
	return fakeConstSig, nil
}

func (f *fakeSig) UnmarshalBinary(buff []byte) error {
	if !bytes.Equal(buff, fakeConstSig) {
		return errors.New("invalid sig")
	}
	return nil
}

func (f *fakeSig) Combine(Signature) Signature {
	return f
}

type fakeCons struct {
}

func (f *fakeCons) Signature() Signature {
	return new(fakeSig)
}

func (f *fakeCons) PublicKey() PublicKey {
	return &fakePublic{true}
}

type fakeNetwork struct{}

func (f *fakeNetwork) RegisterListener(Listener) {
	panic("not implemented yet")
}

func (f *fakeNetwork) Send(Identity, *Packet) error {
	panic("not implemented yet")
}

func fullBitset(level int) BitSet {
	size := int(math.Pow(2, float64(level-1)))
	bs := NewWilffBitset(size)
	for i := 0; i < size; i++ {
		bs.Set(i, true)
	}
	return bs
}

// returns a multisignature from a bitset
func newSig(b BitSet) *MultiSignature {
	return &MultiSignature{
		BitSet:    b,
		Signature: &fakeSig{b.Cardinality(), true},
	}
}

func fullSig(level int) *MultiSignature {
	return newSig(fullBitset(level))
}

func fullSigPair(level int) *sigPair {
	return &sigPair{
		level: byte(level),
		ms:    fullSig(level),
	}
}
