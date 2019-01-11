package main

import (
	"bytes"
	"io"

	"github.com/ConsenSys/handel/bn256"
	crypto "github.com/libp2p/go-libp2p-crypto"
	pb "github.com/libp2p/go-libp2p-crypto/pb"
)

// KeyTypeBN256 bla
const KeyTypeBN256 pb.KeyType = 4

// BN256 type
const BN256 int = 4

func init() {
	/* _, exists := pb.KeyType_name[int32(KeyTypeBN256)]*/
	//if !exists {
	//panic("aie")
	/*}*/
	pb.KeyType_name[int32(KeyTypeBN256)] = "BN256"
	pb.KeyType_value["BN256"] = int32(KeyTypeBN256)
	crypto.PrivKeyUnmarshallers[KeyTypeBN256] = BN256PrivKeyUnmarshaller
	crypto.PubKeyUnmarshallers[KeyTypeBN256] = BN256PubKeyUnmarshaller
	crypto.KeyTypes = append(crypto.KeyTypes, int(KeyTypeBN256))
}

// NewBN256KeyPair returns libp2p adaptor over the bn256 keypair
func NewBN256KeyPair(r io.Reader) (crypto.PrivKey, error) {
	priv, pub, err := bn256.NewKeyPair(r)
	if err != nil {
		return nil, err
	}
	return &bn256Priv{
		SecretKey: priv,
		pub: &bn256Pub{
			PublicKey: pub,
		},
	}, nil
}

type bn256Priv struct {
	*bn256.SecretKey
	pub *bn256Pub
}

func (b *bn256Priv) Bytes() ([]byte, error) {
	return crypto.MarshalPrivateKey(b)
}

func (b *bn256Priv) Equals(key crypto.Key) bool {
	b1, _ := b.Bytes()
	b2, _ := key.(*bn256Priv).MarshalBinary()
	return bytes.Equal(b1, b2)
}

func (b *bn256Priv) Raw() ([]byte, error) {
	return b.MarshalBinary()
}

func (b *bn256Priv) Type() pb.KeyType {
	return KeyTypeBN256
}

func (b *bn256Priv) Sign(msg []byte) ([]byte, error) {
	sig, err := b.SecretKey.Sign(msg, nil)
	if err != nil {
		return nil, err
	}
	return sig.MarshalBinary()
}

func (b *bn256Priv) GetPublic() crypto.PubKey {
	return b.pub
}

type bn256Pub struct {
	*bn256.PublicKey
}

func (b *bn256Pub) Bytes() ([]byte, error) {
	return crypto.MarshalPublicKey(b)
}

func (b *bn256Pub) Equals(k2 crypto.Key) bool {
	b1, _ := b.MarshalBinary()
	b2, _ := k2.(*bn256Pub).MarshalBinary()
	return bytes.Equal(b1, b2)
}

func (b *bn256Pub) Raw() ([]byte, error) {
	return b.MarshalBinary()
}

func (b *bn256Pub) Type() pb.KeyType {
	return KeyTypeBN256
}

func (b *bn256Pub) Verify(data, sig []byte) (bool, error) {
	s := new(bn256.SigBLS)
	if err := s.UnmarshalBinary(sig); err != nil {
		return false, err
	}

	if err := b.VerifySignature(data, s); err != nil {
		return false, err
	}
	return true, nil
}

// BN256PrivKeyUnmarshaller bla
func BN256PrivKeyUnmarshaller(data []byte) (crypto.PrivKey, error) {
	sk := new(bn256.SecretKey)
	return &bn256Priv{SecretKey: sk}, sk.UnmarshalBinary(data)
}

// BN256PubKeyUnmarshaller bla bla bla
func BN256PubKeyUnmarshaller(data []byte) (crypto.PubKey, error) {
	pk := new(bn256.PublicKey)
	return &bn256Pub{PublicKey: pk}, pk.UnmarshalBinary(data)
}
