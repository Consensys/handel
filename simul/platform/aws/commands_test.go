package aws

import (
	"testing"

	"github.com/ConsenSys/handel"
	"github.com/ConsenSys/handel/simul/lib"
	"github.com/stretchr/testify/require"
)

func fakeInstance(publicIP string, ids ...int) Instance {
	var nodes []*lib.Node
	for _, i := range ids {
		id := handel.NewStaticIdentity(int32(i), "", nil)
		node := &lib.Node{nil, id, true, false}
		nodes = append(nodes, node)
	}
	return Instance{PublicIP: &publicIP, Nodes: nodes}
}

func TestOneBinCMDBuilder(t *testing.T) {
	builder := oneBin{syncBasePort: 4000}
	publicIP := "48.224.166.183"
	inst := fakeInstance(publicIP, 1, 2, 3)
	idAndSyncs := builder.startSlave(inst)

	ids := []string{" -id 1", " -id 2", " -id 3"}
	res := idsAndSync{ids: ids, sync: publicIP + ":4000"}

	require.Equal(t, idAndSyncs[0], res)
}

func TestMultiBinCMDBuilder(t *testing.T) {
	builder := multiBin{syncBasePort: 4000}
	publicIP := "48.224.166.183"
	inst := fakeInstance(publicIP, 1, 2, 3)
	idAndSyncs := builder.startSlave(inst)

	res1 := idsAndSync{ids: []string{" -id 1"}, sync: publicIP + ":4001"}
	res2 := idsAndSync{ids: []string{" -id 2"}, sync: publicIP + ":4002"}
	res3 := idsAndSync{ids: []string{" -id 3"}, sync: publicIP + ":4003"}

	res := []idsAndSync{res1, res2, res3}
	require.Equal(t, idAndSyncs, res)
}
