package aws

import (
	"strconv"
)

type cmdbuilder interface {
	startSlave(int Instance) []idsAndSync
}

type idsAndSync struct {
	ids  []string
	sync string
}

type oneBin struct {
	syncBasePort int
}

func (p *oneBin) startSlave(inst Instance) []idsAndSync {
	var iDS []string
	for _, n := range inst.Nodes {
		if !n.Active {
			continue
		}
		id := int(n.ID())
		idsStr := " -id " + strconv.Itoa(id)
		iDS = append(iDS, idsStr)
	}
	sync := GenRemoteAddress(*inst.PublicIP, p.syncBasePort)
	return []idsAndSync{idsAndSync{iDS, sync}}
}

type multiBin struct {
	syncBasePort int
}

func (p *multiBin) startSlave(inst Instance) []idsAndSync {
	var iAS []idsAndSync
	for _, n := range inst.Nodes {
		if !n.Active {
			continue
		}
		id := int(n.ID())
		idsStr := " -id " + strconv.Itoa(id)
		sync := GenRemoteAddress(*inst.PublicIP, p.syncBasePort+id)
		iAS = append(iAS, idsAndSync{[]string{idsStr}, sync})
	}
	return iAS
}

func newCmdbuilder(sameBinary bool, syncBasePort int) cmdbuilder {
	if sameBinary {
		return &oneBin{syncBasePort}
	} else {
		return &multiBin{syncBasePort}
	}
}
