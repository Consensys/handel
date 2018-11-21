package handel

import (
	"errors"
	"math"
)

// partitioner is a generic interface holding the logic used to partition the
// nodes in different buckets.  The only partitioner implemented is
// binTreePartition using binomial tree to partition, as in the original San
// Fermin paper.
type partitioner interface {
	// returns the maximum number of levels this partitioning strategy will use
	// given the list of participants
	MaxLevel() int
	// Returns the size of the set of Identity at this level or an error if
	// level invalid.
	Size(level int) (int, error)
	// IdentitiesAt returns the list of Identity that composes the whole level in
	// this partition scheme.
	IdentitiesAt(level int) ([]Identity, error)
	// PickNextAt returns up to *count* Identity situated at this level. If all
	// identities have been picked already, or if no identities are found at
	// this level, it returns false.
	PickNextAt(level, count int) ([]Identity, bool)
	// Combine takes a list of signature paired with their level and returns all
	// signatures correctly combined according to the partition strategy. The
	// full boolean argument specifies whether the result must be a signature
	// over the FULL bitset or just over the maximum level + 1, to cover all
	// bitset ranges. Typically, one sets full to True when one dispatches the
	// final signature to the application above, which expects a full size
	// bitset. For sending "partial" multi-signatures between handel nodes, full
	// should be set to false, as handel does not send full-size bitsets. All
	// signatures must be valid signatures. The return value can be nil if no
	// sigPairs have been given.
	Combine(sigs []*sigPair, full bool, nbs func(int) BitSet) *sigPair
}

// binTreePartition is a partitioner implementation using a binomial tree
// splitting based on the common length prefix, as in the San Fermin paper.
// It returns new nodes just based on the index alone (no considerations of
// close proximity for example).
type binTreePartition struct {
	// candidatetree computes according to the point of view of this node's id.
	id      int
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
		id:      int(id),
		bitsize: log2(reg.Size()),
		picked:  make(map[int]int),
	}
}

func (c *binTreePartition) MaxLevel() int {
	return log2(c.reg.Size())
}

// IdentitiesAt returns the set of identities that corresponds to the given
// level. It uses the same logic as rangeLevel but returns directly the set of
// identities.
func (c *binTreePartition) IdentitiesAt(level int) ([]Identity, error) {
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
	if level < 0 || level > c.bitsize {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = c.size
	var maxIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= maxIdx && min <= max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		if isSet(uint(c.id), uint(idx)) {
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

// rangeLevelInverse is similar to rangeLevel except that it computes the
// "opposite" group of what rangeLevel returns. It is typically needed to
// compute in what candidate set an ID belongs, or where does a signature in our
// candidate set fits. see CombineF function for one usage.
func (c *binTreePartition) rangeLevelInverse(level int) (min int, max int, err error) {
	if level < 0 || level > c.bitsize+1 {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = c.size
	var maxIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= maxIdx && min <= max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		if isSet(uint(c.id), uint(idx)) {
			min = middle
		} else {
			max = middle
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

func (c *binTreePartition) Combine(sigs []*sigPair, full bool, nbs func(int) BitSet) *sigPair {
	if full {
		return c.combineFull(sigs, nbs)
	}
	return c.combine(sigs, nbs)
}

// combines all all given different-level signatures into one signature
// that has a bitset's size equal to the size of the set of participants,i.e. a
// signature ready to be dispatched to any application.
func (c *binTreePartition) combineFull(sigs []*sigPair, nbs func(int) BitSet) *sigPair {
	if len(sigs) == 0 {
		return nil
	}

	var finalBitSet = nbs(c.reg.Size())

	// set the bits corresponding to the level to the final bitset
	var combineBitSet = func(s *sigPair) {
		min, _, _ := c.rangeLevel(int(s.level))
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			finalBitSet.Set(min+i, bs.Get(i))
		}
	}

	var finalSig = sigs[0].ms.Signature
	combineBitSet(sigs[0])

	var maxLvl = sigs[0].level
	for _, s := range sigs[1:] {
		// combine both signatures
		finalSig = finalSig.Combine(s.ms.Signature)
		combineBitSet(s)
		if s.level > maxLvl {
			maxLvl = s.level
		}
	}
	return &sigPair{
		level: maxLvl,
		ms: &MultiSignature{
			BitSet:    finalBitSet,
			Signature: finalSig,
		},
	}
}

// combines all all given different-level signatures into one signature
// that has a bitset's size equal to the highest level given + 1. The +1 is
// necessary because it covers the whole space in the bitset of all signatures
// together, while the max level only covers its respective signature.
func (c *binTreePartition) combine(sigs []*sigPair, nbs func(int) BitSet) *sigPair {
	if len(sigs) == 0 {
		return nil
	}
	// first, find the range covering all signatures (including potentially
	// missing ones)
	// i.e. if you have level 0 and 2, then the range covering everything is
	// [min, max] where min = minimum of the range of all levels between 0 and 2
	// included, and max = max of the range of all levels between 0 and 2
	// included. Or we can just take the "inverse" range of the next level that
	// covers all levels below :)
	var maxLvl int
	for _, s := range sigs {
		if maxLvl < int(s.level) {
			maxLvl = int(s.level)
		}
	}
	globalMin, globalMax, err := c.rangeLevelInverse(maxLvl + 1)
	if err != nil {
		logf(err.Error())
		return nil
	}

	// create bitset and aggregate signatures
	finalBitSet := nbs(globalMax - globalMin)
	finalSig := sigs[0].ms.Signature

	combine := func(s *sigPair) {
		// compute the offset of this signature compared to the global bitset
		// index
		min, _, _ := c.rangeLevel(int(s.level))
		offset := min - globalMin
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			finalBitSet.Set(offset+i, bs.Get(i))
		}
		finalSig = finalSig.Combine(s.ms.Signature)
	}

	combine(sigs[0])
	for _, s := range sigs[1:] {
		combine(s)
	}

	return &sigPair{
		level: byte(maxLvl + 1),
		ms: &MultiSignature{
			Signature: finalSig,
			BitSet:    finalBitSet,
		},
	}
}
