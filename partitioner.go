package handel

import (
	"errors"
	"math"
)

// partitioner is an interface used to partition the set of nodes in different
// levels. The only partitioner implemented is binTreePartition using binomial
// tree to partition, as in the original San Fermin paper.
type partitioner interface {
	// Returns the size of the set of Identity at this level or an error if
	// level invalid.
	Size(level int) (int, error)
	// RangeAt returns the list of Identity that composes the whole level in
	// this partition scheme.
	RangeAt(level int) ([]Identity, error)
	// PickNextAt returns up to *count* Identity situated at this level. If all
	// identities have been picked already, or if no identities are found at
	// this level, it returns false.
	PickNextAt(level, count int) ([]Identity, bool)
}

// binTreePartition is a partitioner implementation using a binomial tree
// splitting based on the common length prefix, as in the San Fermin paper.
// It returns new nodes just based on the index alone (no considerations of
// close proximity for example).
type binTreePartition struct {
	// candidatetree computes according to the point of view of this node's id.
	id      uint
	bitsize int
	size    int
	reg     Registry
	// mapping for each level of the index of the last node picked for this
	// level
	picked map[int]int
}

// newBinTreePartition returns a binTreePartition using the given ID as its
// anchor point in the ID list, and the given registry.
func newBinTreePartition(id int32, reg Registry) partitioner {
	return &binTreePartition{
		size:    reg.Size(),
		reg:     reg,
		id:      uint(id),
		bitsize: log2(reg.Size()),
		picked:  make(map[int]int),
	}
}

// IdentitiesAt returns the set of identities that corresponds to the given
// level. It uses the same logic as rangeLevel but returns directly the set of
// identities.
func (c *binTreePartition) RangeAt(level int) ([]Identity, error) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return nil, err
	}

	ids, ok := c.reg.Identities(min, max)
	if !ok {
		return nil, errors.New("handel: registry can't find ids in range")
	}
	return ids, nil

}

// rangeLevel returns the range [min,max[ that maps to the set of identity
// comprised at the given level from the point of view of the ID of the
// binTreePartition. At each increasing level, a node should contact nodes from a
// exponentially increasing larger set of nodes, using the binomial tree
// construction as described in the San Fermin paper. Level starts at one and
// ends at the bitsize length. The equality between common prefix length (CPL)
// and level (l) is CPL = bitsize - l.
func (c *binTreePartition) rangeLevel(level int) (min int, max int, err error) {
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

// PickNext returns a set of un-picked identities at the given level, up to
// *count* elements. If no identities could have been picked, it returns false.
func (c *binTreePartition) PickNextAt(level, count int) ([]Identity, bool) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return nil, false
	}

	minPicked, ok := c.picked[level]
	if !ok {
		minPicked = min
	}

	length := max - minPicked
	if length > count {
		max = minPicked + count
	}

	ids, ok := c.reg.Identities(minPicked, max)
	if !ok || length == 0 {
		return nil, false
	}

	c.picked[level] = max
	return ids, true
}

func (c *binTreePartition) Size(level int) (int, error) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return 0, err
	}
	return max - min, nil
}
