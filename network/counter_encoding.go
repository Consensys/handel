package network

import (
	"bytes"
	"io"
	"sync"

	"github.com/ConsenSys/handel"
)

// CounterEncoding is a wrapper around an Encoding that can report how many
// bytes sent and received and implements the monitor.Counter interface
type CounterEncoding struct {
	*sync.RWMutex
	Encoding
	sent int // bytes sent
	rcvd int // bytes received
}

// NewCounterEncoding returns an Encoding that implements the monitor.Counter
// interface
func NewCounterEncoding(e Encoding) *CounterEncoding {
	return &CounterEncoding{Encoding: e, RWMutex: new(sync.RWMutex)}
}

// Encode implements the Encoding interface
func (c *CounterEncoding) Encode(p *handel.Packet, w io.Writer) error {
	var b bytes.Buffer
	combined := io.MultiWriter(w, &b)
	if err := c.Encoding.Encode(p, combined); err != nil {
		return err
	}

	c.Lock()
	c.sent += b.Len()
	c.Unlock()
	return nil
}

// Decode implements the Encoding interface
func (c *CounterEncoding) Decode(r io.Reader) (*handel.Packet, error) {
	var b bytes.Buffer
	var tee = io.TeeReader(r, &b)
	p, err := c.Encoding.Decode(tee)
	if err != nil {
		return nil, err
	}

	c.Lock()
	c.rcvd += b.Len()
	c.Unlock()
	return p, nil
}

// Values implements the monitor.Counter interface
func (c *CounterEncoding) Values() map[string]float64 {
	c.RLock()
	defer c.RUnlock()
	return map[string]float64{
		"sentBytes": float64(c.sent),
		"rcvdBytes": float64(c.rcvd),
	}
}
