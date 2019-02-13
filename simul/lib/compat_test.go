package lib

import (
	"fmt"
	"testing"

	"github.com/ConsenSys/handel/bn256/cf"
	"github.com/stretchr/testify/require"
)

func TestSimulCompatible(t *testing.T) {
	var gen Generator
	gen = bn256.NewConstructor()
	var sc SecretConstructor
	sc = bn256.NewConstructor()
	var cons Constructor
	cons = NewSimulConstructor(bn256.NewConstructor())
	require.NotEqual(t, "", fmt.Sprintf("%v%v%v", gen, sc, cons))
}
