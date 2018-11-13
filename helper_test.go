package handel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func FakeRegistry(size int) Registry {
	ids := make([]Identity, size)
	for i := 0; i < size; i++ {
		ids[i] = &fakeIdentity{int32(i)}
	}
	return NewArrayRegistry(ids)
}

type fakePublic struct{}

func (f *fakePublic) String() string {
	return "fake public key"
}
func (f *fakePublic) VerifySignature([]byte, Signature) error {
	return nil
}
func (f *fakePublic) Combine(PublicKey) PublicKey {
	return f
}

type fakeIdentity struct {
	id int32
}

func (f *fakeIdentity) Address() string      { return fmt.Sprintf("fake-%d", f.id) }
func (f *fakeIdentity) PublicKey() PublicKey { return new(fakePublic) }
func (f *fakeIdentity) ID() int32            { return f.id }
func (f *fakeIdentity) String() string       { return f.Address() }

type fakeSecret struct{}

func (f *fakeSecret) PublicKey() PublicKey {
	return new(fakePublic)
}

func (f *fakeSecret) Sign(msg []byte, rand io.Reader) (Signature, error) {
	return &fakeSig{}, nil
}

var fakeConstSig = []byte{0x01, 0x02, 0x3, 0x04}

type fakeSig struct{}

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

type fakeScheme struct {
	fakeSecret
}

func (f *fakeScheme) Signature() Signature {
	return new(fakeSig)
}

type fakeNetwork struct{}

func (f *fakeNetwork) RegisterListener(Listener) {
	panic("not implemented yet")
}

func (f *fakeNetwork) Send(Identity, *Packet) error {
	panic("not implemented yet")
}
