// package udp enforces each node broadcasts to everybody
package main

import (
	"flag"

	"github.com/ConsenSys/handel/simul/p2p"
)

func main() {
	flag.Parse()
	maker := p2p.AdaptorFunc(MakeUDP)
	p2p.Run(maker)
}
