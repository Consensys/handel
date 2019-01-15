package main

import "github.com/ConsenSys/handel/simul/p2p"

func main() {
	maker := p2p.AdaptorFunc(MakeUDP)
	p2p.Run(maker)
}
