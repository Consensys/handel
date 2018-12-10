package quic

import (
	"crypto/tls"
	"time"

	h "github.com/ConsenSys/handel"
	quic "github.com/lucas-clemente/quic-go"
)

//Dialer is an interface responsible for creating session between two peers
type Dialer interface {
	//startDial create seession between two peers, this is blocking method
	startDial(identity h.Identity, out chan *result)
}

type quicDialer struct {
	handshakeTimeout time.Duration
}

func newQuicDialer(handshakeTimeout time.Duration) Dialer {
	return &quicDialer{handshakeTimeout}
}

func (q quicDialer) startDial(identity h.Identity, out chan *result) {
	tlsCfg := &tls.Config{InsecureSkipVerify: true}

	quicCfg := &quic.Config{HandshakeTimeout: q.handshakeTimeout}
	//Returns session or error of the handshake timeout
	sess, err := quic.DialAddr(identity.Address(), tlsCfg, quicCfg)

	if err != nil {
		out <- &result{identity.ID(), nil, false, err}
		return
	}
	out <- &result{identity.ID(), sess, false, nil}
}
