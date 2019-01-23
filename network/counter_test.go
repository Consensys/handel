package network

import (
	"bytes"
	"testing"

	"github.com/ConsenSys/handel"
	"github.com/stretchr/testify/require"
)

func TestCounterEncoding(t *testing.T) {
	var medium bytes.Buffer

	actual := NewGOBEncoding()
	counter := NewCounterEncoding(actual)

	require.True(t, counter.Values()["sentBytes"] == 0.0)
	require.True(t, counter.Values()["rcvdBytes"] == 0.0)

	toSend := &handel.Packet{
		Origin:   156,
		Level:    8,
		MultiSig: []byte("History repeats itself, first as tragedy, second as farce."),
	}

	require.NoError(t, counter.Encode(toSend, &medium))
	require.True(t, counter.Values()["sentBytes"] > 0.0)

	read, err := counter.Decode(&medium)
	require.NoError(t, err)

	require.Equal(t, toSend, read)
	require.True(t, counter.Values()["rcvdBytes"] > 0.0)
}
