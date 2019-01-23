package lib

import (
	"fmt"
	"math/rand"
)

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
	poffline := offline
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
	for i < total || offline > 0 {
		// put one ID in one platform at a time, round robin fashion
		for _, plat := range plats {
			s := plat.String()
			// find the first non allocated node
			list := out[s]
			for idx, ni := range list {
				if ni.ID != -1 {
					// already allocated
					continue
				}
				list[idx].ID = i
				if i == nextOffline && offline > 0 {
					list[idx].Active = false
					nextOffline = (nextOffline + bucketOffline) % total
					offline--
					/*if nextOffline <= i || nextOffline >= total {*/
					//fmt.Printf("i %d - nextOffline %d\n", i, nextOffline)
					//panic("internal error")
					/*}*/
				} else {
					list[idx].Active = true
				}
				i++
				break
			}
		}
	}

	verifyAllocation(out, total, poffline)
	return out
}

// RoundRandomOffline  uses RoundRobin allocator to setups the alive nodes but chooses
// randomly offline nodes in a round robin way for each platforms
type RoundRandomOffline struct {
	Allocator
}

// NewRoundRandomOffline returns a RoundRandomOffline allocator
func NewRoundRandomOffline() Allocator {
	return &RoundRandomOffline{
		Allocator: new(RoundRobin),
	}
}

// Allocate implements the Allocator interface
func (r *RoundRandomOffline) Allocate(plats []Platform, total, offline int) map[string][]*NodeInfo {

	poffline := offline
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
	i := 0
	// allocate all ids
	for i < total {
		// put one ID in one platform at a time, round robin fashion
		for _, plat := range plats {
			s := plat.String()
			// find the first non allocated node
			list := out[s]
			for idx, ni := range list {
				if ni.ID != -1 {
					// already allocated
					continue
				}
				list[idx].ID = i
				list[idx].Active = true
				i++
				break
			}
		}
	}

	for offline > 0 {
		// put one offline per platform randomly at a time
		for _, plat := range plats {
			s := plat.String()
			list := out[s]
			n := len(list)
			var i int
			// search first index where we have an active one randomly
			for i = rand.Intn(n); list[i].Active == false; i = rand.Intn(n) {
			}
			list[i].Active = false
			offline--
			if offline == 0 {
				break
			}
		}
	}
	verifyAllocation(out, total, poffline)
	return out
}

func verifyAllocation(allocation map[string][]*NodeInfo, total, offline int) {
	dead := 0
	live := 0
	for k, list := range allocation {
		fmt.Printf("\t[+] plat %s: ", k)
		localDead, localLive := 0, 0
		for _, node := range list {
			//fmt.Printf("%d (%v)- ", node.ID, node.Active)
			if node.ID < 0 {
				panic("id not allocated")
			}
			if !node.Active {
				localDead++
			} else {
				localLive++
			}
		}
		fmt.Printf(" %d alive %d dead", localLive, localDead)
		fmt.Printf("\n")
		dead += localDead
		live += localLive
	}

	if dead != offline {
		fmt.Printf("DEAD %d - WANTED %d\n", dead, offline)
		panic("wrong number of dead nodes")
	}
	if dead+live != total {
		panic("wrong number of total nodes")
	}
}
