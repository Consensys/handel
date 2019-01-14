package main

import "github.com/ConsenSys/handel/simul/p2p"

func main() {
	p2p.Run(p2p.AdaptorFunc(MakeP2P))
}
