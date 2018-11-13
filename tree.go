package handel

import (
	"errors"
	"math"
)

// candidateTree have different methods manipulating the logical binomial tree a
// Handel node uses. It can compute the index ranges corresponding to a given
// level, saves wich candidate have already been contacted at a given level
// TODO: potentially put a generic "contact strategy" interface that can deal with
// different ways to select peers at a given level. For example, if we know
// additional information such as the distance, we way wish to use the closest
// nodes first.
type candidateTree struct {
	// candidatetree computes according to the point of view of this node's id.
	id      uint
	bitsize int
	size    int
	reg     Registry
}

// newCandidateTree returns a candidateTree using the given ID as its anchor
// point in the ID list, and the given registry.
func newCandidateTree(id int32, reg Registry) *candidateTree {
	return &candidateTree{
		size:    reg.Size(),
		reg:     reg,
		id:      uint(id),
		bitsize: log2(reg.Size()),
	}
}

// IdentitiesAt returns the set of identities that corresponds to the given
// level. It uses the same logic as RangeAt but returns directly the set of
// identities.
func (c *candidateTree) IdentitiesAt(level int) ([]Identity, error) {
	min, max, err := c.RangeAt(level)
	if err != nil {
		return nil, err
	}

	ids, ok := c.reg.Identities(min, max)
	if !ok {
		return nil, errors.New("handel: registry can't find ids in range")
	}
	return ids, nil

}

// RangeAt returns the range [min,max[ that maps to the set of identity
// comprised at the given level from the point of view of the ID of the
// candidateTree. At each increasing level, a node should contact nodes from a
// exponentially increasing larger set of nodes, using the binomial tree
// construction as described in the San Fermin paper. Level starts at one and
// ends at the bitsize length. The equality between common prefix length (CPL)
// and level (l) is CPL = bitsize - l.
func (c *candidateTree) RangeAt(level int) (min int, max int, err error) {
	if level < 1 || level > c.bitsize {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = c.size
	var maxIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= maxIdx && min <= max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		if isSet(c.id, uint(idx)) {
			// we inverse the order at the given CPL to get the candidate set.
			// Otherwise we would get the same set as c.id is in.
			if idx == maxIdx {
				max = middle
			} else {
				min = middle
			}
		} else {
			// same inversion here
			if idx == maxIdx {
				min = middle
			} else {
				max = middle
			}
		}
		if max == min {
			break
		}

		if max-1 == 0 || min == c.size {
			break
		}
	}
	return min, max, nil
}
