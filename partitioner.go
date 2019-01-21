package handel

import (
	"errors"
	"fmt"
	"math"
)

// Partitioner is a generic interface holding the logic used to partition the
// nodes in different buckets.  The only Partitioner implemented is
// binTreePartition using binomial tree to partition, as in the original San
// Fermin paper.
type Partitioner interface {
	// returns the maximum number of levels this partitioning strategy will use
	// given the list of participants
	MaxLevel() int
	// Returns the size of the set of Identity at this level
	Size(level int) int

	// Levels returns the list of level ids Handel must run on.
	// It does not return the level 0 since that represents the personal contributions of
	// the Handel node itself.
	// If the levels is empty (it happens when the number of nodes is not a power of two),
	//  it is not included in the returned list
	Levels() []int
	// IdentitiesAt returns the list of Identity that composes the whole level in
	// this partition scheme.
	IdentitiesAt(level int) ([]Identity, error)

	// IndexAtLevel returns the index inside the given level of the given global
	// ID. The returned index is usable inside a bitset for the same level.
	IndexAtLevel(globalID int32, level int) (int, error)

	// Combine takes a list of signature paired with their level and returns all
	// signatures correctly combined according to the partition strategy.
	// All signatures must be valid signatures. The return value can be nil if no
	// incomingSig have been given.It returns a MultiSignature whose's BitSet's
	// size is equal to the size of the level given in parameter + 1. The +1 is
	// there because it is a combined signature, therefore, encompassing all
	// signatures of levels up to the given level included.
	Combine(sigs []*incomingSig, level int, nbs func(int) BitSet) *MultiSignature
	// CombineFull is similar to Combine but it returns the full multisignature
	// whose bitset's length is equal to the size of the registry. This length
	// corresponds to the MaxLevel() + 1 - but this level is not considered a
	// "valid" level from a Handel perspective.
	CombineFull(sigs []*incomingSig, nbs func(int) BitSet) *MultiSignature
}

// binomialPartitioner is a partitioner implementation using a binomial tree
// splitting based on the common length prefix, as in the San Fermin paper.
// It returns new nodes just based on the index alone (no considerations of
// close proximity for example).
type binomialPartitioner struct {
	// candidatetree computes according to the point of view of this node's id.
	id      int
	bitsize int
	size    int
	reg     Registry
	// mapping for each level of the index of the last node picked for this
	// level
	picked map[int]int
	logger Logger
}

// NewBinPartitioner returns a binTreePartition using the given ID as its
// anchor point in the ID list, and the given registry.
func NewBinPartitioner(id int32, reg Registry, logger Logger) Partitioner {
	return &binomialPartitioner{
		size:    reg.Size(),
		reg:     reg,
		id:      int(id),
		bitsize: log2(reg.Size()),
		picked:  make(map[int]int),
		logger:  logger,
	}
}

func (c *binomialPartitioner) MaxLevel() int {
	return log2(c.reg.Size())
}

// IdentitiesAt returns the set of identities that corresponds to the given
// level. It uses the same logic as rangeLevel but returns directly the set of
// identities.
func (c *binomialPartitioner) IdentitiesAt(level int) ([]Identity, error) {
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

func (c *binomialPartitioner) Levels() []int {
	var levels []int
	for i := 1; i <= c.MaxLevel(); i++ {
		_, _, err := c.rangeLevel(i)
		if err != nil {
			continue
		}
		levels = append(levels, i)
	}
	return levels
}

func (c *binomialPartitioner) IndexAtLevel(globalID int32, level int) (int, error) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return 0, err
	}
	id := int(globalID)
	if id < min || id >= max {
		err := fmt.Errorf("globalID outside level's range. id=%d, min=%d, max=%d, level=%d", id, min, max, level)
		c.logger.Warn(err) // If it happens it's either a bug either an attack from a byzantine node
		return 0, err
	}
	return id - min, nil
}

// errEmptyLevel is returned when a range for a requested level is empty. This
// can happen is the number of nodes is not a perfect power of two.
var errEmptyLevel = errors.New("empty level")

// rangeLevel returns the range [min,max[ that maps to the set of identity
// comprised at the given level from the point of view of the ID of the
// binTreePartition. At each increasing level, a node should contact nodes from
// a exponentially increasing larger set of nodes, using the binomial tree
// construction as described in the San Fermin paper. Level starts at 0 (same
// node) and ends at the bitsize length + 1 (whole ID range).
// It returns errEmptyLevel if the range corresponding to the given level is
// empty.It returns an error if the level requested is out of bound.
func (c *binomialPartitioner) rangeLevel(level int) (min int, max int, err error) {
	if level < 0 || level > c.bitsize+1 {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = pow2(log2(c.size))
	var inverseIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= inverseIdx && idx >= 0 && min < max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		//fmt.Printf("id %d, idx %d, inverseIdx %d, bitsize %d, min %d, middle %d, max %d\n", c.id, idx, inverseIdx, c.bitsize, min, middle, max)

		if isSet(uint(c.id), uint(idx)) {
			// we inverse the order at the given CPL to get the candidate set.
			// Otherwise we would get the same set as c.id is in (as in
			// rangeLevelInverse)
			if idx == inverseIdx {
				max = middle
			} else {
				min = middle
			}
		} else {
			// same inversion here
			if idx == inverseIdx {
				min = middle
			} else {
				max = middle
			}
		}

	}

	//  >= because the minimum index is inclusive
	if min >= c.size {
		return 0, 0, errEmptyLevel
	}

	// > because the maximum index is exclusive
	if max > c.size {
		max = c.size
	}

	return min, max, nil
}

// rangeLevelInverse is similar to rangeLevel except that it computes the
// "opposite" group of what rangeLevel returns. It is typically needed to
// compute in what candidate set an ID belongs, or where does a signature in our
// candidate set fits. see CombineF function for one usage. It returns an error
// if the given level is out of bound.
func (c *binomialPartitioner) rangeLevelInverse(level int) (min int, max int, err error) {
	if level < 0 || level > c.bitsize+1 {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = pow2(log2(c.size))
	var maxIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= maxIdx && idx >= 0 && min < max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		//fmt.Printf("id %d, idx %d, inverseIdx %d, bitsize %d, min %d, middle %d, max %d\n", c.id, idx, maxIdx, c.bitsize, min, middle, max)

		if isSet(uint(c.id), uint(idx)) {
			min = middle
		} else {
			max = middle
		}
	}

	if max > c.size {
		max = c.size
	}
	return min, max, nil

}

func (c *binomialPartitioner) Size(level int) int {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		if err == errEmptyLevel {
			return 0
		}
		panic(err)
	}
	return max - min
}

func (c *binomialPartitioner) Combine(sigs []*incomingSig, level int, nbs func(int) BitSet) *MultiSignature {
	if len(sigs) == 0 {
		return nil
	}

	for _, s := range sigs {
		if int(s.level) > level {
			logf("invalid combination of signature / requested level")
			return nil
		}
	}

	// taking the "rangeInverse" gives us the range covering all signatures
	// with a level inferior than "level" - it's the range nodes at the
	// corresponding candidate set expect to receive.
	globalMin, globalMax, err := c.rangeLevelInverse(level)
	if err != nil {
		logf(err.Error())
		return nil
	}
	size := globalMax - globalMin
	bitset := nbs(size)
	//fmt.Printf("\t -- Combine(lvl %d) => min %d max %d -> size %d\n", level, globalMin, globalMax, size)
	combined := func(s *incomingSig, final BitSet) {
		// compute the offset of this signature compared to the global bitset
		// index
		min, _, _ := c.rangeLevel(int(s.level))
		offset := min - globalMin
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			final.Set(offset+i, bs.Get(i))
		}
	}

	return c.combineSize(sigs, bitset, combined)
}

func (c *binomialPartitioner) CombineFull(sigs []*incomingSig, nbs func(int) BitSet) *MultiSignature {
	if len(sigs) == 0 {
		return nil
	}
	var finalBitSet = nbs(c.reg.Size())

	// set the bits corresponding to the level to the final bitset
	var combineBitSet = func(s *incomingSig, final BitSet) {
		min, _, _ := c.rangeLevel(int(s.level))
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			final.Set(min+i, bs.Get(i))
		}
	}
	return c.combineSize(sigs, finalBitSet, combineBitSet)
}

// combineSize combines all given signature witht he combine function on the
// bitset using `bs`
func (c *binomialPartitioner) combineSize(sigs []*incomingSig, bs BitSet, combine func(*incomingSig, BitSet)) *MultiSignature {

	var finalSig = sigs[0].ms.Signature
	combine(sigs[0], bs)

	for _, s := range sigs[1:] {
		// combine both signatures
		finalSig = finalSig.Combine(s.ms.Signature)
		combine(s, bs)
	}
	return &MultiSignature{
		BitSet:    bs,
		Signature: finalSig,
	}
}

// combines all all given different-level signatures into one signature
// that has a bitset's size equal to the highest level given + 1. The +1 is
// necessary because it covers the whole space in the bitset of all signatures
// together, while the max level only covers its respective signature.
func (c *binomialPartitioner) combine(sigs []*incomingSig, nbs func(int) BitSet) *incomingSig {
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

	combine := func(s *incomingSig) {
		// compute the offset of this signature compared to the global bitset
		// index
		min, _, _ := c.rangeLevel(int(s.level))
		offset := min - globalMin
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			finalBitSet.Set(offset+i, bs.Get(i))
		}
	}

	combine(sigs[0])
	for _, s := range sigs[1:] {
		combine(s)
		finalSig = finalSig.Combine(s.ms.Signature)
	}

	return &incomingSig{
		level: byte(maxLvl + 1),
		ms: &MultiSignature{
			Signature: finalSig,
			BitSet:    finalBitSet,
		},
	}
}
