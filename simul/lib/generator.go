package lib

import (
	"crypto/rand"

	h "github.com/ConsenSys/handel"
)

// GenerateNodes create the necessary key pair & identites out of the given addresses.
// The IDs will be created sequentially from 0.
func GenerateNodes(cons Constructor, addresses []string) []*Node {
	nodes := make([]*Node, len(addresses))
	for i, addr := range addresses {
		sec, pub := cons.KeyPair(rand.Reader)
		id := h.NewStaticIdentity(int32(i), addr, pub)
		nodes[i] = &Node{SecretKey: sec, Identity: id}
	}
	return nodes
}

// WriteAll writes down all the given nodes to the specified URI with the given
// parser.
func WriteAll(nodes []*Node, p NodeParser, uri string) {
	records := make([]*NodeRecord, len(nodes))
	for i, n := range nodes {
		rec, err := n.ToRecord()
		if err != nil {
			panic(err)
		}
		records[i] = rec
	}
	if err := p.Write(uri, records); err != nil {
		panic(err)
	}
}
