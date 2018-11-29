package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func csvContent() [][]string {
	csv := [][]string{
		{"id", "port", "addr", "pubKey"},
		{"0", "3000", "127.0.0.1", ""},
		{"1", "3001", "127.0.0.1", ""},
		{"2", "3002", "127.0.0.1", ""},
	}
	return csv
}

func csvCorrupted() [][]string {
	csv := [][]string{
		{"id", "port", "addr", "pubKey"},
		{"0", "3000", "127.0.0.1", ""},
		{"x", "3001", "127.0.0.1", ""},
		{"2", "3002", "127.0.0.1", ""},
	}
	return csv
}

func TestCSVRegistry(t *testing.T) {
	csv := csvContent()
	reg, port, err := makeRegistry(csv, 1, NewEmptyPublicKeyCsvParser())
	require.Nil(t, err)
	require.Equal(t, port, 3001)
	require.Equal(t, reg.Size(), 3)
	id, _ := reg.Identity(2)
	require.Equal(t, id.Address(), "127.0.0.1:3002")
}

func TestCorruptedCSVRegistry(t *testing.T) {
	csv := csvCorrupted()
	_, _, err := makeRegistry(csv, 1, NewEmptyPublicKeyCsvParser())
	require.NotNil(t, err)
}
