package house

/*
func TestP2P(t *testing.T) {

	n := 50
	thr := 50
	var opts = map[string]string{"Fanout": "10", "Period": "100ms"}
	maker := p2p.AdaptorFunc(MakeGossip)
	//maker = p2p.WithConnector(maker)
	/*maker = p2p.WithPostFunc(maker, func(r handel.Registry, nodes []p2p.Node) {*/
//var wg sync.WaitGroup
//for _, n := range nodes {
//wg.Add(1)
//go func(n *P2PNode) {
//n.WaitAllSetup()
//wg.Done()
//}(n.(*P2PNode))
//}
//wg.Wait()
//})

//	test.Aggregators(t, n, thr, maker, opts, lib.GetFreeUDPPort)

//}
