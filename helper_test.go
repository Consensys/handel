package handel

import (
	"bytes"
	"errors"
	"io"
)

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

type fakeIdentity struct{}

func (f *fakeIdentity) Address() string      { return "fake identity" }
func (f *fakeIdentity) PublicKey() PublicKey { return new(fakePublic) }

type fakeSecret struct{}

func (f *fakeSecret) PublicKey() PublicKey {
	return new(fakePublic)
}

func (f *fakeSecret) Sign(msg []byte, rand io.Reader) (Signature, error) {
	return &fakeSig{}, nil
}

var sig = []byte{0x01, 0x02, 0x3, 0x04}

type fakeSig struct{}

func (f *fakeSig) MarshalBinary() ([]byte, error) {
	return sig, nil
}

func (f *fakeSig) UnmarshalBinary(buff []byte) error {
	if !bytes.Equal(buff, sig) {
		return errors.New("invalid sig")
	}
	return nil
}

func (f *fakeSig) Combine(Signature) Signature {
	return f
}
