package lib

import "fmt"

// Platform represents the platform where multiple Handel nodes can run. It can
// be a process for localhost platform's or EC2 instance for aws.
type Platform interface {
	String() string
}

// NodeInfo is the output of the allocator. The allocator only tries to assign
// the ID and the status, not the physical network address since that is
// dependent on the chosen platform.
type NodeInfo struct {
	ID      int
	Active  bool
	Address string
}

// Allocator allocates *total* Handel instances on *len(plats)* platform, where
// *offline* instances will be set as offline.
type Allocator interface {
	Allocate(plats []Platform, total, offline int) map[string][]*NodeInfo
}

// RoundRobin allocates the nodes in a round robin fashion as to minimise the
// number of nodes on each platforms.
type RoundRobin struct{}

// Allocate implements the Allocator2 interface
func (r *RoundRobin) Allocate(plats []Platform, total, offline int) map[string][]*NodeInfo {
	n := len(plats)
	out := make(map[string][]*NodeInfo)
	instPerPlat, rem := Divmod(total, n)
	for i := 0; i < n; i++ {
		// add instPerPlat instances to the i-th platform
		s := plats[i].String()
		for j := 0; j < instPerPlat; j++ {
			out[s] = append(out[s], &NodeInfo{ID: -1})
		}
		if rem > 0 {
			out[s] = append(out[s], &NodeInfo{ID: -1})
			rem--
		}
	}

	// dispatch the IDs online then offline by round robin
	bucketOffline := total
	if offline != 0 {
		bucketOffline, _ = Divmod(total, offline)
	}
	i := 0
	nextOffline := 0
	// allocate all ids
	for i < total {
		// put one ID in one platform at a time, roundrobin fashion
		for _, plat := range plats {
			s := plat.String()
			// find the first non allocated node
			list := out[s]
			for idx, ni := range list {
				if ni.ID != -1 {
					// already allocated
					continue
				}
				var status = true
				if i == nextOffline && offline > 0 {
					status = false
					nextOffline = (i + bucketOffline + 1) % total
					offline--
				}
				list[idx].ID = i
				list[idx].Active = status
				i++
				break
			}
			out[s] = list
		}
	}

	for k, list := range out {
		fmt.Printf("\t[+] plat %s: ", k)
		for _, node := range list {
			fmt.Printf("%d (%v)- ", node.ID, node.Active)
		}
		fmt.Printf("\n")
	}
	return out
}
