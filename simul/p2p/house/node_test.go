package house

/*
func TestHouseHandel(t *testing.T) {
	n := 40
	fanout := 5
	period := 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, nodes, err := FakeSetup(ctx, n, fanout, period)
	if err != nil {
		panic(err)
	}

	printAll := func(topic string, ns []*Node) {
		for _, n := range ns {
			n.Lock()
			fmt.Printf("%d for topic %s has:", n.Identity().ID(), topic)
			state, exists := n.topics[topic]
			if !exists {
				fmt.Printf("NOTHING")
			}
			fmt.Printf("\n")
			//state.Lock()
			got := make([]bool, len(ns))
			for i := 0; i < len(ns); i++ {
				_, ex := state.gossips[int32(i)]
				if ex {
					got[i] = true
				}
			}
			//state.Unlock()
			for i, h := range got {
				fmt.Printf("\t%d: %v", i, h)
				if i+1%6 == 0 {
					fmt.Printf("\n")
				}
			}
			fmt.Printf("\n")
			n.Unlock()
		}
	}
	for i, n := range nodes {
		buff := []byte("Hello My Gossip")
		topic := handelTopic
		gossip := &Gossip{
			ID:    n.Identity().ID(),
			Topic: topic,
			Msg:   buff,
		}
		fmt.Println("\n\nDIFFUSING for node ", i, " topic : ", gossip.Topic)
		n.Gossip(gossip)
		for j, n2 := range nodes {
			if j == i {
				continue
			}
			select {
			case <-n2.NextTopic(topic):
			case <-time.After(1*time.Second + period*5):
				printAll(topic, nodes)
				panic("aie")
			}
		}
		for _, n2 := range nodes {
			n2.StopGossip(topic)
		}
	}
}
func TestHouseDifferentTopic(t *testing.T) {
	n := 40
	fanout := 5
	period := 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, nodes, err := FakeSetup(ctx, n, fanout, period)
	if err != nil {
		panic(err)
	}

	printAll := func(topic string, ns []*Node) {
		for _, n := range ns {
			n.Lock()
			fmt.Printf("%d for topic %s has:", n.Identity().ID(), topic)
			state, exists := n.topics[topic]
			if !exists {
				fmt.Printf("NOTHING")
			}
			fmt.Printf("\n")
			//state.Lock()
			got := make([]bool, len(ns))
			for i := 0; i < len(ns); i++ {
				_, ex := state.gossips[int32(i)]
				if ex {
					got[i] = true
				}
			}
			//state.Unlock()
			for i, h := range got {
				fmt.Printf("\t%d: %v", i, h)
				if i+1%6 == 0 {
					fmt.Printf("\n")
				}
			}
			fmt.Printf("\n")
			n.Unlock()
		}
	}
	topicFromID := func(id int32) string {
		return fmt.Sprintf("test-%d", id)
	}
	for i, n := range nodes {
		buff := []byte("Hello My Gossip")
		topic := topicFromID(n.Identity().ID())
		gossip := &Gossip{
			ID:    n.Identity().ID(),
			Topic: topic,
			Msg:   buff,
		}
		fmt.Println("\n\nDIFFUSING for node ", i, " topic : ", gossip.Topic)
		n.Gossip(gossip)
		for j, n2 := range nodes {
			if j == i {
				continue
			}
			select {
			case <-n2.NextTopic(topic):
			case <-time.After(period * 5):
				printAll(topic, nodes)
				panic("aie")
			}
		}
		for _, n2 := range nodes {
			n2.StopGossip(topic)
		}
	}
}

func FakeSetup(ctx context.Context, n, fanout int, period time.Duration) ([]*lib.Node, []*Node, error) {
	base := 20000
	addresses := make([]string, n)
	for i := 0; i < n; i++ {
		port := base + i
		address := "127.0.0.1:" + strconv.Itoa(port)
		addresses[i] = address
	}
	cons := lib.NewSimulConstructor(bn256.NewConstructor())
	ctx = context.WithValue(ctx, p2p.CtxKey("Constructor"), cons)
	nodes := lib.GenerateNodes(cons, addresses)
	nodeList := lib.NodeList(nodes)
	registry := nodeList.Registry()

	hnodes := make([]*Node, n)
	var err error
	for i := 0; i < n; i++ {
		node := nodes[i]
		hnodes[i], err = NewNode(node, registry, fanout, period)
		if err != nil {
			panic(err)
		}
	}

	return nodes, hnodes, nil
}
*/
