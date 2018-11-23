package bn256

import (
	"crypto/rand"
	"testing"
	"time"

	h "github.com/ConsenSys/handel"
	"github.com/stretchr/testify/require"
)

func TestHandel(t *testing.T) {
	n := 16
	msg := []byte("Peaches and Cream")
	secretKeys := make([]h.SecretKey, n)
	cons := NewConstructor()
	for i := 0; i < n; i++ {
		k, err := NewSecretKey(nil)
		require.NoError(t, err)
		secretKeys[i] = k
	}
	test := h.NewTest(secretKeys, cons, msg)
	test.Start()
	defer test.Stop()

	select {
	case <-test.WaitCompleteSuccess():
	case <-time.After(1 * time.Second):
		t.FailNow()
	}
}

func TestSign(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	sk, err := NewSecretKey(reader)
	require.NoError(t, err)

	sig, err := sk.Sign(msg, nil)
	require.NoError(t, err)

	pk := sk.PublicKey()
	err = pk.VerifySignature(msg, sig)
	require.NoError(t, err)
}

func TestCombine(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	sk1, err := NewSecretKey(reader)
	require.NoError(t, err)

	sk2, err := NewSecretKey(reader)
	require.NoError(t, err)

	pk1 := sk1.PublicKey()
	pk2 := sk2.PublicKey()
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
