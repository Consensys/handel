package scenarios

import (
	"testing"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/stretchr/testify/require"
)

func TestNodeInc(t *testing.T) {
	defaultConf := lib.Config{
		Network:     "udp",
		Curve:       "bn256",
		Encoding:    "gob",
		MonitorPort: 10000,
		MaxTimeout:  "2m",
		Retrials:    1,
	}

	handel := &lib.HandelConfig{
		Period:      "20ms",
		UpdateCount: 1,
		NodeCount:   10,
		Timeout:     "100ms",
	}

	nodesInc := NewNodeInc(defaultConf, handel, 99, 4, 0, CalcThreshold80)
	conf := nodesInc.Generate()
	require.Equal(t, len(conf.Runs), 24)

	for i, r := range conf.Runs {
		nodes := (i + 1) * 4
		require.Equal(t, *r.Handel, *handel)
		require.Equal(t, r.Nodes, nodes)
		require.Equal(t, r.Processes, (i + 1))
		require.Equal(t, r.GetThreshold(), CalcThreshold80(nodes))
	}
	require.Equal(t, len(defaultConf.Runs), 0)
}

func TestCalcThreshold(t *testing.T) {
	require.Equal(t, CalcThreshold80(10), 8)
	require.Equal(t, CalcThreshold80(9), 8)
	require.Equal(t, CalcThreshold80(8), 7)

	require.Equal(t, CalcThreshold51(10), 6)
	require.Equal(t, CalcThreshold51(9), 5)
	require.Equal(t, CalcThreshold51(8), 5)
}
