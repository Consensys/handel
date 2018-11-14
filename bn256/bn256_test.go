package bn256

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	sk, err := NewSecretKey(reader)
	require.NoError(t, err)

	sig, err := sk.Sign(msg, nil)
	require.NoError(t, err)

	pk := sk.Public()
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

	pk1 := sk1.Public()
	pk2 := sk2.Public()
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
