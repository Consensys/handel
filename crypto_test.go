package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiSignatureMarshalling(t *testing.T) {
	bs := NewWilffBitset(10)
	bs.Set(1, true)
	bs.Set(9, true)

	sig := new(fakeSig)

	ms := &MultiSignature{BitSet: bs, Signature: sig}
	buff, err := ms.MarshalBinary()
	require.NoError(t, err)

	ms2 := new(MultiSignature)
	err = ms2.Unmarshal(buff, new(fakeSig), new(WilffBitSet))
	require.NoError(t, err)

}
