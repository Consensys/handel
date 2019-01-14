package aws

import (
	"fmt"

	"github.com/ConsenSys/handel/simul/lib"
)

//Instance represents EC2 Amazon instance
type Instance struct {
	// EC2 ID
	ID *string
	// IP Visible to the outside world
	PublicIP *string
	// State: running, pending, stopped
	State *string
	//EC2 Instance region
	region string
	// EC2 Instance TAG
	Tag string

	Nodes []*lib.Node
}

//Manager manages group of EC2 instances
type Manager interface {
	// Instances lists avaliable instances in any state
	Instances() []Instance
	// RefreshInstances populates the instance list and updates instances status
	RefreshInstances() ([]Instance, error)
	// StartInstances starts all avaliable instances and populates the instance list,
	// blocks until all instances are in "running" state
	StartInstances() error
	// StopInstances stops all avaliable instances
	StopInstances() error
}

const base = 3000

// GenRemoteAddresses generates n * 2 addresses: one for handel, one for the sync
func GenRemoteAddresses(instances []Instance) ([]string, []string) {
	n := len(instances)
	var addresses = make([]string, 0, n)
	var syncs = make([]string, 0, n)
	for _, i := range instances {
		addr1 := GenRemoteAddress(*i.PublicIP, base)
		addr2 := GenRemoteAddress(*i.PublicIP, base+1)
		addresses = append(addresses, addr1)
		syncs = append(syncs, addr2)
	}
	return addresses, syncs
}

// GenRemoteAddress generates Node address
func GenRemoteAddress(ip string, port int) string {
	addr := fmt.Sprintf("%s:%d", ip, port)
	return addr
}

type info struct {
	id     int
	active bool
}

// UpdateInstances bla
func UpdateInstances(actives []int, total int, instances []*Instance, cons lib.Constructor) {
	nodesPerInstance := make([][]info, len(instances))
	instID := 0
	for id := 0; id < total; id++ {
		var active bool
		if isContained(actives, id) {
			active = true
		}
		info := info{id, active}
		nodesPerInstance[instID] = append(nodesPerInstance[instID], info)
		instID++
		if instID >= len(instances) {
			instID = 0
		}
	}

	for i, inst := range instances {
		UpdateInstance(nodesPerInstance[i], inst, cons)
	}
}

func isContained(arr []int, v int) bool {
	for _, v2 := range arr {
		if v2 == v {
			return true
		}
	}
	return false
}

// UpdateInstance bla
func UpdateInstance(nodes []info, instances *Instance, cons lib.Constructor) {
	var ls []*lib.Node
	for i, n := range nodes {
		addr1 := GenRemoteAddress(*instances.PublicIP, base+i)
		node := lib.GenerateNode(cons, n.id, addr1)
		node.Active = n.active
		ls = append(ls, node)
	}
	instances.Nodes = ls
}

// WaitUntilAllInstancesRunning blocks until all instances are
// in the "running" state
func WaitUntilAllInstancesRunning(a Manager, delay func()) (int, error) {
	allRunning := allInstancesRunning(a.Instances())
	if allRunning {
		return 0, nil
	}
	tries := 0
	for {
		tries++
		delay()
		allInstances, err := a.RefreshInstances()
		if err != nil {
			return tries, err
		}
		allRunning = allInstancesRunning(allInstances)
		if allRunning {
			return tries, nil
		}
	}
}

func allInstancesRunning(instances []Instance) bool {
	okInstances := 0
	for _, inst := range instances {
		if (*inst.State) == running {
			okInstances++
			if okInstances >= len(instances) {
				return true
			}
		}
	}
	return false
}

func instanceToInstanceID(instances []Instance) []*string {
	var ids []*string
	for _, inst := range instances {
		ids = append(ids, inst.ID)
	}
	return ids
}
