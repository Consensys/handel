package quic

import (
	"crypto/tls"
	"time"

	h "github.com/ConsenSys/handel"
	quic "github.com/lucas-clemente/quic-go"
)

// dialer is an interface responsible for creating session between two peers,
// quicDialer implements dialer for the quic protocol
type dialer interface {
	//startDial create seession between two peers, this is blocking method
	startDial(identity h.Identity, out chan *result)
}

type quicDialer struct {
	handshakeTimeout   time.Duration
	insecureSkipVerify bool
	serverName         string
}

func newQuicDialer(handshakeTimeout time.Duration, serverName string) dialer {
	return &quicDialer{handshakeTimeout, false, serverName}
}

func newInsecureQuicDialer(handshakeTimeout time.Duration) dialer {
	return &quicDialer{handshakeTimeout, true, ""}

}

func (q quicDialer) startDial(identity h.Identity, out chan *result) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: q.insecureSkipVerify,
		ServerName:         q.serverName,
	}
	quicCfg := &quic.Config{HandshakeTimeout: q.handshakeTimeout}
	//Returns session or error of the handshake timeout
	sess, err := quic.DialAddr(identity.Address(), tlsCfg, quicCfg)

	if err != nil {
		out <- &result{identity.ID(), nil, false, err}
		return
	}
	out <- &result{identity.ID(), sess, false, nil}
}
