package bn256

import (
	"crypto/rand"
	"testing"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/stretchr/testify/require"
)

func TestHandel(t *testing.T) {
	n := 101
	config := h.DefaultConfig(n)
	msg := []byte("Peaches and Cream")
	secretKeys := make([]h.SecretKey, n)
	pubKeys := make([]h.PublicKey, n)
	cons := NewConstructor()
	for i := 0; i < n; i++ {
		sec, pub, err := NewKeyPair(nil)
		require.NoError(t, err)
		secretKeys[i] = sec
		pubKeys[i] = pub
	}
	test := h.NewTest(secretKeys, pubKeys, cons, msg, config)
	//test.SetOfflineNodes(15, 25, 8)
	//test.SetThreshold(n - 4)
	test.Start()
	defer test.Stop()

	select {
	case <-test.WaitCompleteSuccess():
	case <-time.After(100 * time.Second):
		t.FailNow()
	}
}

func TestSign(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	sk, pk, err := NewKeyPair(reader)
	require.NoError(t, err)

	sig, err := sk.Sign(msg, nil)
	require.NoError(t, err)

	err = pk.VerifySignature(msg, sig)
	require.NoError(t, err)
}

func TestCombine(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	sk1, pk1, err := NewKeyPair(reader)
	require.NoError(t, err)

	sk2, pk2, err := NewKeyPair(reader)
	require.NoError(t, err)

	require.NotEqual(t, pk1.String(), pk2.String())

	sig1, err := sk1.Sign(msg, nil)
	require.NoError(t, err)
	require.NoError(t, pk1.VerifySignature(msg, sig1))

	sig2, err := sk2.Sign(msg, nil)
	require.NoError(t, err)
	require.NoError(t, pk2.VerifySignature(msg, sig2))

	sig3 := sig1.Combine(sig2)
	pk3 := pk1.Combine(pk2)
	require.NoError(t, pk3.VerifySignature(msg, sig3))
}

func TestMarshalling(t *testing.T) {

	sk, pk, err := NewKeyPair(nil)
	require.NoError(t, err)

	buffSK, err := sk.MarshalBinary()
	require.NoError(t, err)

	buffPK, err := pk.MarshalBinary()
	require.NoError(t, err)

	cons := NewConstructor()

	sk2 := cons.SecretKey()
	err = sk2.(*SecretKey).UnmarshalBinary(buffSK)
	require.NoError(t, err)

	pk2 := cons.PublicKey()
	err = pk2.(*PublicKey).UnmarshalBinary(buffPK)
	require.NoError(t, err)
}
