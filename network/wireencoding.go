package network

import (
	"io"

	h "github.com/ConsenSys/handel"
)

// Encoding abstract the wire encoding format
type Encoding interface {
	Encode(*h.Packet, io.Writer) error
	Decode(r io.Reader) (*h.Packet, error)
}
