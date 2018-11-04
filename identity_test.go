package handel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type fakePublic struct{}

func (f *fakePublic) String() string {
	return "fake public key"
}
func (f *fakePublic) VerifySignature([]byte, MultiSignature) error {
	return nil
}
func (f *fakePublic) Combine(PublicKey) PublicKey {
	return f
}

type fakeIdentity struct{}

func (f *fakeIdentity) Address() string      { return "fake identity" }
func (f *fakeIdentity) PublicKey() PublicKey { return new(fakePublic) }

type registryTest struct {
	reg          func() Registry
	expectedSize int
	// single identity test
	getIdx      int
	getFound    bool
	getIdentity Identity
	// range test
	from       int
	to         int
	found      bool
	identities []Identity
}

func TestRegistryArray(t *testing.T) {
	n := 10
	ids := make([]Identity, n, n)
	for i := 0; i < n; i++ {
		ids[i] = new(fakeIdentity)
	}
	registry := NewArrayRegistry(ids)
	nf := func() Registry { return registry }
	var tests = []registryTest{
		{
			nf, 10, 1, true, ids[1], 0, 3, true, ids[0:3],
		},
		{
			nf, 10, -1, false, nil, 0, 11, false, nil,
		},
	}
	testRegistryTests(t, tests)
}

func testRegistryTests(t *testing.T, tests []registryTest) {
	for _, test := range tests {
		registry := test.reg()
		require.Equal(t, test.expectedSize, registry.Size())

		id, found := registry.Identity(test.getIdx)
		require.Equal(t, test.getFound, found)
		if found {
			require.Equal(t, test.getIdentity, id)
		}

		ids, rangeFound := registry.Identities(test.from, test.to)
		require.Equal(t, test.found, rangeFound)
		if rangeFound {
			require.Equal(t, test.identities, ids)
		}
	}
}
