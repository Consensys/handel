package network

import (
	"encoding/gob"
	"io"

	h "github.com/ConsenSys/handel"
)

type gobEncoding struct {
}

// NewGOBEncoding crates instance of Encoding interface backed by gob
func NewGOBEncoding() Encoding {
	return &gobEncoding{}
}

// Encode implements the Encoding interface
func (g gobEncoding) Encode(packet *h.Packet, w io.Writer) error {
	enc := gob.NewEncoder(w)
	err := enc.Encode(packet)
	return err
}

// Decode implements the Encoding interface
func (g gobEncoding) Decode(r io.Reader) (*h.Packet, error) {
	var packet h.Packet
	//Decode gob encoded packet
	dec := gob.NewDecoder(r)
	err := dec.Decode(&packet)
	return &packet, err
}
